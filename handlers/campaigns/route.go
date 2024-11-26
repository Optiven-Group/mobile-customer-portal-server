package campaigns

import "github.com/gin-gonic/gin"

func RegisterCampaignsRoutes(r *gin.RouterGroup) {
	r.GET("/monthly-campaign", GetMonthlyCampaign)
}
