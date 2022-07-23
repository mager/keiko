package handler

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	"github.com/mager/go-opensea/opensea"
	"github.com/mager/sweeper/database"
	ens "github.com/wealdtech/go-ens/v3"
)

type NFT struct {
	Name     string     `json:"name"`
	TokenID  string     `json:"tokenId"`
	ImageURL string     `json:"imageUrl"`
	Traits   []NFTTrait `json:"traits"`
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

type CollectionResp struct {
	Name     string    `json:"name"`
	Floor    float64   `json:"floor"`
	Slug     string    `json:"slug"`
	Thumb    string    `json:"thumb"`
	NumOwned int       `json:"numOwned"`
	Updated  time.Time `json:"updated"`
	NFTs     []NFT     `json:"nfts"`
}

// GetAddressResp is the response for the GET /v2/info endpoint
type GetAddressResp struct {
	Address     string           `json:"address"`
	Collections []CollectionResp `json:"collections"`
	TotalETH    float64          `json:"totalETH"`
	ETHPrice    float64          `json:"ethPrice"`
	ENSName     string           `json:"ensName"`
	UpdatedAt   time.Time        `json:"updatedAt"`
	User        User             `json:"user"`
}

// getAddress is the route handler for the GET /address/{address} endpoint
func (h *Handler) getAddress(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		address = strings.ToLower(mux.Vars(r)["address"])
		ensName string
		// c       = cache.New(5*time.Minute, 10*time.Minute)
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
		sort.Slice(resp.Collections[:], func(i, j int) bool {
			return resp.Collections[i].Floor > resp.Collections[j].Floor
		})

		resp.UpdatedAt = user.Wallet.UpdatedAt
	} else {
		h.logger.Info("User not found in database, returning", "address", address)
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

func (h *Handler) adaptWalletToCollectionResp(wallet database.Wallet) ([]CollectionResp, float64) {
	var (
		resp           = []CollectionResp{}
		collections    = h.dbClient.Client.Collection("collections")
		dbCollections  = []database.Collection{}
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
	for _, docsnap := range docsnaps {
		var collection database.Collection
		err := docsnap.DataTo(&collection)
		if err != nil {
			h.logger.Error(err)
		}
		dbCollections = append(dbCollections, collection)
	}

	for _, c := range wallet.Collections {
		numOwned := len(c.NFTs)
		floor := h.adaptFloor(dbCollections, c)
		resp = append(resp, CollectionResp{
			Name:     c.Name,
			Slug:     c.Slug,
			Thumb:    c.ImageURL,
			NFTs:     adaptWalletNFTsToCollectionRespNFTs(c.NFTs),
			Floor:    floor,
			NumOwned: numOwned,
		})
		totalETH += float64(numOwned) * floor
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
		})
	}

	return resp
}

func (h *Handler) adaptFloor(collections []database.Collection, wc database.WalletCollection) float64 {
	var floor float64

	for _, collection := range collections {
		if collection.Slug == wc.Slug {
			floor = collection.Floor
		}
	}

	return floor
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
	}
}

// New
func (h *Handler) getNFTCollection(ctx context.Context, address string) []CollectionResp {
	var (
		openseaAssets      = make([]opensea.Asset, 0)
		collectionsMap     = make(map[string]CollectionResp)
		collectionSlugDocs = make([]*firestore.DocumentRef, 0)
	)

	// Fetch the user's collections & NFTs from OpenSea
	openseaAssets, err := h.os.GetAssets(address)
	if err != nil {
		h.logger.Error(err)
		return []CollectionResp{}
	}

	// Create a list of wallet collections
	for _, asset := range openseaAssets {
		// If we do have a collection for this asset, add to it
		if _, ok := collectionsMap[asset.Collection.Slug]; ok {
			w := collectionsMap[asset.Collection.Slug]
			w.NFTs = append(w.NFTs, NFT{
				Name:     asset.Name,
				ImageURL: asset.ImageURL,
				TokenID:  asset.TokenID,
			})
			collectionsMap[asset.Collection.Slug] = w
			continue
		} else {
			// If we don't have a collection for this asset, create it
			collectionsMap[asset.Collection.Slug] = CollectionResp{
				Name:  asset.Collection.Name,
				Slug:  asset.Collection.Slug,
				Thumb: asset.Collection.ImageURL,
				NFTs: []NFT{{
					Name:     asset.Name,
					TokenID:  asset.TokenID,
					ImageURL: asset.ImageURL,
				}},
			}
		}

	}

	// Construct a wallet object
	var collections = make([]CollectionResp, 0)
	for _, collection := range collectionsMap {
		collections = append(collections, collection)
		collectionSlugDocs = append(collectionSlugDocs, h.dbClient.Client.Collection("collections").Doc(collection.Slug))
	}

	// Add the floor prices
	var (
		slugToFloorMap = make(map[string]float64)
		slugsToAdd     = make([]string, 0)
	)

	docsnaps, err := h.dbClient.Client.GetAll(h.ctx, collectionSlugDocs)
	if err != nil {
		h.logger.Error(err)
	}

	for _, docsnap := range docsnaps {
		if !docsnap.Exists() {
			h.logger.Infow("Collection not found in Firestore", "collection", docsnap.Ref.ID)
			slugsToAdd = append(slugsToAdd, docsnap.Ref.ID)
		} else {
			slugToFloorMap[docsnap.Ref.ID] = docsnap.Data()["floor"].(float64)
		}
	}

	// Call sweeper to update add new collections
	if len(slugsToAdd) > 0 {
		go h.sweeper.AddCollections(slugsToAdd)
	}

	for i, collection := range collections {
		collection.Floor = slugToFloorMap[collection.Slug]
		collections[i] = collection
	}

	return collections
}
