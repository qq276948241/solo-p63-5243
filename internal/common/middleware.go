package common

import (
	"strings"

	"github.com/gin-gonic/gin"
)

type UserContext struct {
	UserID   uint
	Username string
	Role     string
}

const UserContextKey = "user"

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			Unauthorized(c, "未提供认证令牌")
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			Unauthorized(c, "认证令牌格式错误")
			c.Abort()
			return
		}

		claims, err := ParseToken(parts[1])
		if err != nil {
			Unauthorized(c, "认证令牌无效或已过期")
			c.Abort()
			return
		}

		userCtx := &UserContext{
			UserID:   claims.UserID,
			Username: claims.Username,
			Role:     claims.Role,
		}

		c.Set(UserContextKey, userCtx)
		c.Next()
	}
}

func RoleAuth(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userCtx, exists := c.Get(UserContextKey)
		if !exists {
			Unauthorized(c, "用户未认证")
			c.Abort()
			return
		}

		user := userCtx.(*UserContext)
		hasRole := false
		for _, role := range roles {
			if user.Role == role {
				hasRole = true
				break
			}
		}

		if !hasRole {
			Forbidden(c, "权限不足")
			c.Abort()
			return
		}

		c.Next()
	}
}

func GetCurrentUser(c *gin.Context) *UserContext {
	userCtx, exists := c.Get(UserContextKey)
	if !exists {
		return nil
	}
	return userCtx.(*UserContext)
}
