package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/pkg/auth"
	"github.com/jylc/cloudserver/pkg/cluster"
	"github.com/jylc/cloudserver/pkg/serializer"
	"strconv"
)

func SlaveRPCSignRequired(nodePool cluster.Pool) gin.HandlerFunc {
	return func(c *gin.Context) {
		nodeID, err := strconv.ParseUint(c.GetHeader(auth.CrHeaderPrefix+"Node-Id"), 10, 64)
		if err != nil {
			c.JSON(200, serializer.ParamErr("未知的主机节点ID", err))
			c.Abort()
			return
		}

		slaveNode := nodePool.GetNodeByID(uint(nodeID))
		if slaveNode == nil {
			c.JSON(200, serializer.ParamErr("未知的主机节点ID", err))
			c.Abort()
			return
		}

		SignRequired(slaveNode.MasterAuthInstance())(c)
	}
}
