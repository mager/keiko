package handler

import (
	"encoding/json"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	"github.com/mager/keiko/utils"
	"github.com/mager/sweeper/database"
	ens "github.com/wealdtech/go-ens/v3"
)

type NFT struct {
	Name     string     `json:"name"`
	TokenID  string     `json:"tokenId"`
	ImageURL string     `json:"imageUrl"`
	Traits   []NFTTrait `json:"traits"`
	Floor    float64    `json:"floor"`
}

type NFTTrait struct {
	Name       string `json:"name"`
	Value      string `json:"value"`
	OpenSeaURL string `json:"openSeaURL"`
}

// Collection represents floor report collection
type Collection struct {
	Name            string  `json:"name"`
	FloorPrice      float64 `json:"floorPrice"`
	OneDayChange    float64 `json:"oneDayChange"`
	ImageURL        string  `json:"imageUrl"`
	NFTs            []NFT   `json:"nfts"`
	OpenSeaURL      string  `json:"openSeaURL"`
	OwnedAssetCount int     `json:"ownedAssetCount"`
	UnrealizedValue float64 `json:"unrealizedValue"`
}

type CollectionStat struct {
	Slug       string  `json:"slug"`
	FloorPrice float64 `json:"floorPrice"`
}

// InfoReq is a request to /info
type InfoReq struct {
	Address string `json:"address"`
	SkipBQ  bool   `json:"skipBQ"`
}

type AddressCollection struct {
	Name string `json:"name"`
	// Value is the combined value of all NFTs in the collection
	Value float64 `json:"value"`
	// Floor is the collection floor price
	Floor    float64   `json:"floor"`
	Slug     string    `json:"slug"`
	Thumb    string    `json:"thumb"`
	NumOwned int       `json:"numOwned"`
	Updated  time.Time `json:"updated"`
	NFTs     []NFT     `json:"nfts"`
}

// GetAddressResp is the response for the GET /v2/info endpoint
type GetAddressResp struct {
	Address     string              `json:"address"`
	Collections []AddressCollection `json:"collections"`
	TotalETH    float64             `json:"totalETH"`
	TotalUSD    float64             `json:"totalUSD"`
	ENSName     string              `json:"ensName"`
	UpdatedAt   time.Time           `json:"updatedAt"`
	User        User                `json:"user"`
	Updating    bool                `json:"updating"`
}

// getAddress is the route handler for the GET /address/{address} endpoint
func (h *Handler) getAddress(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		address = strings.ToLower(mux.Vars(r)["address"])
		ensName string
	)

	// Make sure that the request includes an address
	if address == "" {
		http.Error(w, "you must include an ETH address in the request", http.StatusBadRequest)
		return
	}

	// Validate address
	if !common.IsHexAddress(address) {
		// Fetch address from ENS if it's not a valid address
		ensName = address
		address = h.infuraClient.GetAddressFromENSName(address)
		if address == "" {
			http.Error(w, "you must include a valid ETH address in the request", http.StatusBadRequest)
			return
		}
	}

	var (
		resp = GetAddressResp{
			Address: address,
		}
		ensNameChan = make(chan string)
	)

	// Get ENS Name
	if ensName == "" {
		go h.asyncGetENSNameFromAddress(address, ensNameChan)
		resp.ENSName = <-ensNameChan
	} else {
		resp.ENSName = ensName
	}

	// Convert to lowercase
	address = strings.ToLower(address)

	// Check if the user exists in the database first
	user, err := h.fetchUser(address)
	if err == nil {
		resp.User = h.adaptUser(user)
		h.logger.Infow("User found in database", "address", address)
		resp.Collections, resp.TotalETH = h.adaptWalletToCollectionResp(user.Wallet)
		if len(user.Wallet.Collections) == 0 {
			resp.Updating = true
		}
		sort.Slice(resp.Collections[:], func(i, j int) bool {
			return resp.Collections[i].Value > resp.Collections[j].Value
		})

		resp.UpdatedAt = user.Wallet.UpdatedAt
	} else {
		h.logger.Info("User not found in database, returning", "address", address)
	}

	// Fetch ETH price
	ethPriceUSD := h.cs.GetETHPrice()

	resp.TotalUSD = utils.AdaptTotalUSD(resp.TotalETH, ethPriceUSD)

	// Filter out 0ETH collections
	if user.Settings.HideZeroETHCollections {
		var filteredCollections []AddressCollection
		for _, c := range resp.Collections {
			if c.Floor > 0 {
				filteredCollections = append(filteredCollections, c)
			}

		}
		resp.Collections = filteredCollections

	}

	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) asyncGetENSNameFromAddress(address string, rc chan string) {
	domain, err := ens.ReverseResolve(h.infuraClient.Client, common.HexToAddress(address))
	if err != nil {
		h.logger.Error(err)
		rc <- ""
	}

	rc <- domain
}

func (h *Handler) adaptWalletToCollectionResp(wallet database.Wallet) ([]AddressCollection, float64) {
	var (
		resp           = []AddressCollection{}
		collections    = h.dbClient.Client.Collection("collections")
		collectionDocs = make([]*firestore.DocumentRef, 0)
		totalETH       float64
	)

	for _, collection := range wallet.Collections {
		collectionDocs = append(collectionDocs, collections.Doc(collection.Slug))
	}

	// Fetch collections from Firestore
	docsnaps, err := h.dbClient.Client.GetAll(h.ctx, collectionDocs)
	if err != nil {
		h.logger.Error(err)
	}

	h.logger.Infof("%d collections found in Firestore", len(docsnaps))

	for _, c := range wallet.Collections {
		numOwned := len(c.NFTs)
		nfts := adaptWalletNFTsToCollectionRespNFTs(c.NFTs)
		value := h.adaptValue(nfts)
		resp = append(resp, AddressCollection{
			Name:     c.Name,
			Slug:     c.Slug,
			Thumb:    c.ImageURL,
			NFTs:     nfts,
			Floor:    h.adaptFloor(c),
			Value:    value,
			NumOwned: numOwned,
		})
		totalETH += value
	}

	// Round to 3 decimal places
	totalETH = math.Round(totalETH*1000) / 1000

	return resp, totalETH
}

func adaptWalletNFTsToCollectionRespNFTs(walletNFTs []database.WalletAsset) []NFT {
	var resp = []NFT{}

	for _, walletNFT := range walletNFTs {
		resp = append(resp, NFT{
			Name:     walletNFT.Name,
			TokenID:  walletNFT.TokenID,
			ImageURL: walletNFT.ImageURL,
			Floor:    walletNFT.Floor,
		})
	}

	return resp
}

func (h *Handler) adaptValue(nfts []NFT) float64 {
	var val float64

	for _, nft := range nfts {
		val += nft.Floor
	}

	return math.Round(val*100) / 100
}

func (h *Handler) adaptFloor(wc database.WalletCollection) float64 {
	return math.Round(wc.Floor*100) / 100
}

func (h *Handler) adaptUser(user database.User) User {
	return User{
		Name:        user.Name,
		Photo:       user.Photo,
		ENSName:     user.ENSName,
		Collections: user.Collections,
		Slug:        user.Slug,
		Twitter:     user.Twitter,
		IsFren:      user.IsFren,
		DiscordID:   user.DiscordID,
		Settings:    user.Settings,
	}
}
