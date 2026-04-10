package handlers

import (
	"net/http"
	"swapngo-backend/internal/bizs"
	"swapngo-backend/internal/ws"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)
type PriceHandler interface {
	HandleWS(ctx *gin.Context)
}

type priceHandler struct {
	priceBiz bizs.PriceBiz
	hub *ws.Hub
}

func NewPriceHandler(priceBiz bizs.PriceBiz, hub *ws.Hub) PriceHandler {
	return &priceHandler{priceBiz: priceBiz, hub: hub}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h *priceHandler) HandleWS(ctx *gin.Context) {
	// 从中间件获取 userID
	userID := ctx.GetString("user_id")
	
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		return
	}

	h.hub.Register(userID, conn)
	
	// 启动该用户的价格推送协程 (这里可以注入 Biz 层)
	h.priceBiz.StartPushing(userID)
}