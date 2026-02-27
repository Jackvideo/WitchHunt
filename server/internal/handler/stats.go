package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackvidyu/witchhunt/server/internal/model"
	"gorm.io/gorm"
)

func GetUserStats(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := uint(c.GetFloat64("user_id"))

		var stats model.UserStatsRow
		db.Model(&model.GameParticipant{}).
			Select(`
				COUNT(*) as total_games,
				SUM(CASE WHEN won = 1 THEN 1 ELSE 0 END) as wins,
				SUM(CASE WHEN won = 0 THEN 1 ELSE 0 END) as losses,
				SUM(CASE WHEN is_witch = 1 THEN 1 ELSE 0 END) as witch_games,
				SUM(CASE WHEN is_witch = 1 AND won = 1 THEN 1 ELSE 0 END) as witch_wins,
				SUM(CASE WHEN is_witch = 0 THEN 1 ELSE 0 END) as villager_games,
				SUM(CASE WHEN is_witch = 0 AND won = 1 THEN 1 ELSE 0 END) as villager_wins
			`).
			Where("user_id = ? AND is_bot = 0", uid).
			Scan(&stats)

		c.JSON(http.StatusOK, stats)
	}
}

func GetUserHistory(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		uid := uint(c.GetFloat64("user_id"))

		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
		if page < 1 {
			page = 1
		}
		if size < 1 || size > 50 {
			size = 20
		}
		offset := (page - 1) * size

		var participantIDs []uint
		db.Model(&model.GameParticipant{}).
			Select("game_record_id").
			Where("user_id = ? AND is_bot = 0", uid).
			Order("game_record_id DESC").
			Offset(offset).Limit(size).
			Pluck("game_record_id", &participantIDs)

		if len(participantIDs) == 0 {
			c.JSON(http.StatusOK, gin.H{"records": []any{}, "page": page, "size": size, "total": 0})
			return
		}

		var total int64
		db.Model(&model.GameParticipant{}).Where("user_id = ? AND is_bot = 0", uid).Count(&total)

		var records []model.GameRecord
		db.Preload("Participants").
			Where("id IN ?", participantIDs).
			Order("created_at DESC").
			Find(&records)

		type historyItem struct {
			model.GameRecord
			YourRole string `json:"your_role"`
			Won      bool   `json:"won"`
			Alive    bool   `json:"alive"`
		}

		var items []historyItem
		for _, r := range records {
			item := historyItem{GameRecord: r}
			for _, p := range r.Participants {
				if p.UserID == uid {
					if p.IsWitch {
						item.YourRole = "witch"
					} else {
						item.YourRole = "villager"
					}
					item.Won = p.Won
					item.Alive = p.Alive
					break
				}
			}
			items = append(items, item)
		}

		c.JSON(http.StatusOK, gin.H{
			"records": items,
			"page":    page,
			"size":    size,
			"total":   total,
		})
	}
}
