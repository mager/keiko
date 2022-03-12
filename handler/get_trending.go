package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"

	"cloud.google.com/go/firestore"
	"github.com/mager/keiko/utils"
	"github.com/mager/sweeper/database"
	"google.golang.org/api/iterator"
)

type GetTrendingResp struct {
	TopHighestFloor []database.Collection `json:"topHighestFloor"`
	TopWeeklyVolume []database.Collection `json:"topWeeklyVolume"`
}

func (h *Handler) getTrending(w http.ResponseWriter, r *http.Request) {
	var (
		ctx                     = context.TODO()
		resp                    = GetTrendingResp{}
		collections             = h.dbClient.Client.Collection("collections")
		highestFloorCollections = make([]database.Collection, 0)
		highestFloorCounter     = 0
	)

	// Fetch collections with the highest floor price
	highestFloorIter := collections.Documents(ctx)

	for {
		doc, err := highestFloorIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			h.logger.Errorf("Error fetching collections: %v", err)
			break
		}

		thumb, ok := doc.Data()["thumb"].(string)
		if !ok {
			thumb = ""
		}

		sevenDayVolume, ok := doc.Data()["7d"].(float64)
		if !ok {
			sevenDayVolume = 0.0
		}

		// Only add collections with a weekly volume of over 1 ETH
		if sevenDayVolume > 1.0 {
			highestFloorCollections = append(highestFloorCollections, database.Collection{
				Name:           doc.Data()["name"].(string),
				Slug:           doc.Data()["slug"].(string),
				Thumb:          thumb,
				SevenDayVolume: utils.RoundFloat(sevenDayVolume, 2),
				Floor:          utils.RoundFloat(doc.Data()["floor"].(float64), 2),
			})
		}
	}

	// Sort highest floor collections by floor price
	sort.Slice(highestFloorCollections[:], func(i, j int) bool {
		return highestFloorCollections[i].Floor > highestFloorCollections[j].Floor
	})

	// Only add the first 25 items to the response
	for _, collection := range highestFloorCollections {
		if highestFloorCounter < 25 {
			resp.TopHighestFloor = append(resp.TopHighestFloor, collection)
		}
		highestFloorCounter++
	}

	// Fetch collections with the highest 7d weekly volume
	highestWeeklyVolumeIter := collections.OrderBy("7d", firestore.Desc).Limit(25).Documents(ctx)
	for {
		doc, err := highestWeeklyVolumeIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			h.logger.Errorf("Error fetching collections: %v", err)
			break
		}

		thumb, ok := doc.Data()["thumb"].(string)
		if !ok {
			thumb = ""
		}

		sevenDayVolume, ok := doc.Data()["7d"].(float64)
		if !ok {
			sevenDayVolume = 0.0
		}

		resp.TopWeeklyVolume = append(resp.TopWeeklyVolume, database.Collection{
			Name:           doc.Data()["name"].(string),
			Slug:           doc.Data()["slug"].(string),
			Thumb:          thumb,
			SevenDayVolume: utils.RoundFloat(sevenDayVolume, 2),
			Floor:          utils.RoundFloat(doc.Data()["floor"].(float64), 2),
		})
	}

	json.NewEncoder(w).Encode(resp)
}
