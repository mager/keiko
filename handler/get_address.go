package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/ethereum/go-ethereum/common"
	"github.com/gorilla/mux"
	"github.com/mager/keiko/bigquery"
	"github.com/mager/keiko/opensea"
	"github.com/mager/keiko/utils"
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
	User        database.User    `json:"user"`
}

// getAddress is the route handler for the GET /address/{address} endpoint
func (h *Handler) getAddress(w http.ResponseWriter, r *http.Request) {
	var (
		err     error
		req     InfoReq
		address = mux.Vars(r)["address"]
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
	} else {
		address = common.HexToAddress(address).String()
	}

	var (
		collections        = make([]opensea.OpenSeaCollectionCollection, 0)
		nfts               = make([]opensea.OpenSeaAsset, 0)
		collectionSlugDocs = make([]*firestore.DocumentRef, 0)
		resp               = GetAddressResp{
			Address: address,
		}
		ethPrice        float64
		totalETH        float64
		nftsChan        = make(chan []opensea.OpenSeaAsset)
		collectionsChan = make(chan []opensea.OpenSeaCollectionCollection)
		ethPriceChan    = make(chan float64)
		ensNameChan     = make(chan string)
	)

	// Get ENS Name
	if ensName == "" {
		go h.asyncGetENSNameFromAddress(address, ensNameChan)
		resp.ENSName = <-ensNameChan
	} else {
		resp.ENSName = ensName
	}

	// Check if the user exists in the database first
	user, err := h.getUser(address)
	if err != nil {
		// // Fetch the user's collections & NFTs from OpenSea
		go h.asyncGetOpenSeaCollections(address, w, collectionsChan)
		collections = <-collectionsChan

		go h.asyncGetOpenSeaAssets(address, w, nftsChan)
		nfts = <-nftsChan

		// Get ETH price
		go h.asyncGetETHPrice(ethPriceChan)
		ethPrice = <-ethPriceChan
		resp.ETHPrice = ethPrice

		var slugToOSCollectionMap = make(map[string]opensea.OpenSeaCollectionCollection)
		for _, collection := range collections {
			collectionSlugDocs = append(collectionSlugDocs, h.database.Collection("collections").Doc(collection.Slug))
			slugToOSCollectionMap[collection.Slug] = collection
		}

		// Check if the user's collections are in our database
		docsnaps, err := h.database.GetAll(h.ctx, collectionSlugDocs)
		if err != nil {
			h.logger.Error(err)
			return
		}

		var docSnapMap = make(map[string]database.Collection)
		var collectionRespMap = make(map[string]CollectionResp)
		for _, ds := range docsnaps {
			if ds.Exists() {
				numOwned := slugToOSCollectionMap[ds.Ref.ID].OwnedAssetCount
				floor := ds.Data()["floor"].(float64)
				// This is for the response
				collectionRespMap[ds.Ref.ID] = CollectionResp{
					Name:     ds.Data()["name"].(string),
					Floor:    floor,
					Slug:     ds.Ref.ID,
					Updated:  ds.Data()["updated"].(time.Time),
					Thumb:    slugToOSCollectionMap[ds.Ref.ID].ImageURL,
					NumOwned: numOwned,
					NFTs:     h.getNFTsForCollection(ds.Ref.ID, nfts),
				}
				// This is for Firestore
				docSnapMap[ds.Ref.ID] = database.Collection{
					Floor:   floor,
					Name:    ds.Data()["name"].(string),
					Slug:    ds.Ref.ID,
					Updated: ds.Data()["updated"].(time.Time),
				}

				totalETH += utils.RoundFloat(float64(numOwned)*floor, 4)
			}
		}

		for _, collection := range collections {
			// Check docSnapMap to see if collection slug is in there
			if _, ok := docSnapMap[collection.Slug]; ok {
				resp.Collections = append(resp.Collections, collectionRespMap[collection.Slug])
			} else {
				// Otherwise, add it to the database
				go h.sweeper.AddCollection(collection.Slug)
			}
		}
		resp.TotalETH = totalETH
	} else {
		resp.User = user
		resp.Collections, resp.TotalETH = h.adaptWalletToCollectionResp(user.Wallet)
	}

	if !req.SkipBQ {
		bigquery.RecordRequestInBigQuery(
			h.bq.DatasetInProject("floor-report-327113", "info"),
			h.logger,
			address,
		)
	}

	sort.Slice(resp.Collections[:], func(i, j int) bool {
		return resp.Collections[i].Floor > resp.Collections[j].Floor
	})

	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) getNFTsForCollection(slug string, nfts []opensea.OpenSeaAsset) []NFT {
	var result []NFT
	for _, nft := range nfts {
		if nft.Collection.Slug == slug {
			result = append(result, NFT{
				Name:     nft.Name,
				TokenID:  nft.TokenID,
				ImageURL: nft.ImageThumbnailURL,
				Traits:   getNFTTraits(nft),
			})
		}
	}
	return result
}

func getNFTTraits(asset opensea.OpenSeaAsset) []NFTTrait {
	var result []NFTTrait
	if len(asset.Traits) == 0 {
		return result
	}
	for _, trait := range asset.Traits {
		traitValueStr := getNFTTraitValue(trait.Value)
		nftTrait := NFTTrait{
			Name:       trait.TraitType,
			Value:      traitValueStr,
			OpenSeaURL: getOpenSeaTraitURL(asset, trait),
		}
		result = append(result, nftTrait)
	}
	return result
}

func getNFTTraitValue(t interface{}) string {
	switch t.(type) {
	case string:
		return t.(string)
	case int:
		return fmt.Sprintf("%d", t.(int))
	case float64:
		return fmt.Sprintf("%f", t.(float64))
	default:
		return ""
	}
}

func getOpenSeaTraitURL(asset opensea.OpenSeaAsset, trait opensea.OpenSeaAssetTrait) string {
	return fmt.Sprintf(
		"https://opensea.io/collection/%s?search[stringTraits][0][name]=%s&search[stringTraits][0][values][0]=%s",
		asset.Collection.Slug,
		url.QueryEscape(trait.TraitType),
		url.QueryEscape(getNFTTraitValue(trait.Value)),
	)
}

// asyncGetOpenSeaCollections gets the collections from OpenSea
func (h *Handler) asyncGetOpenSeaCollections(address string, w http.ResponseWriter, rc chan []opensea.OpenSeaCollectionCollection) {
	collections, err := h.os.GetAllCollectionsForAddress(address)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rc <- collections
}

// asyncGetOpenSeaAssets gets the assets for the given address
func (h *Handler) asyncGetOpenSeaAssets(address string, w http.ResponseWriter, rc chan []opensea.OpenSeaAsset) {
	nfts, err := h.os.GetAllAssetsForAddress(address)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rc <- nfts
}

// asyncGetETHPrice gets the ETH price from the ETH price API
func (h *Handler) asyncGetETHPrice(rc chan float64) {
	rc <- h.cs.GetETHPrice()
}

func (h *Handler) asyncGetENSNameFromAddress(address string, rc chan string) {
	domain, err := ens.ReverseResolve(h.infuraClient.Client, common.HexToAddress(address))
	if err != nil {
		h.logger.Error(err)
		rc <- ""
	}

	rc <- domain
}

func (h *Handler) getUser(address string) (database.User, error) {
	// Fetch user from Firestore
	var user database.User

	docsnap, err := h.database.Collection("users").Doc(address).Get(h.ctx)
	if err != nil {
		h.logger.Error(err)
		return user, err
	}

	if docsnap.Exists() {
		err = docsnap.DataTo(&user)
		if err != nil {
			h.logger.Error(err)
		}
	} else {
		h.logger.Info("User not found in Firestore")
		return user, err
	}

	return user, nil
}

func (h *Handler) adaptWalletToCollectionResp(wallet database.Wallet) ([]CollectionResp, float64) {
	var (
		resp           = []CollectionResp{}
		collections    = h.database.Collection("collections")
		dbCollections  = []database.Collection{}
		collectionDocs = make([]*firestore.DocumentRef, 0)
		totalETH       float64
	)

	for _, collection := range wallet.Collections {
		collectionDocs = append(collectionDocs, collections.Doc(collection.Slug))
	}

	// Fetch collections from Firestore
	docsnaps, err := h.database.GetAll(h.ctx, collectionDocs)
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
			h.logger.Infow("Found floor for collection", "collection", collection.Slug, "floor", floor)
		}
	}

	return floor
}