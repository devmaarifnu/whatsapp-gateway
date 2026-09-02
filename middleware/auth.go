package middleware

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"whatsapp-gateway/repository"
)

func Auth(tokenRepo *repository.TokenRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		raw := strings.TrimPrefix(header, "Bearer ")

		// 1. Plain token
		record, err := tokenRepo.FindByToken(raw)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}

		// 2. SHA256 hashed
		if record == nil {
			hashed := fmt.Sprintf("%x", sha256.Sum256([]byte(raw)))
			record, err = tokenRepo.FindByToken(hashed)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "db error"})
				return
			}
		}

		// 3. Bcrypt
		if record == nil {
			candidates, err := tokenRepo.FindByHashType("bcrypt")
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "db error"})
				return
			}
			for _, candidate := range candidates {
				if bcrypt.CompareHashAndPassword([]byte(candidate.Token), []byte(raw)) == nil {
					record = candidate
					break
				}
			}
		}

		if record == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		if record.ExpiresAt != nil && record.ExpiresAt.Before(time.Now()) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "token expired"})
			return
		}

		c.Next()
	}
}

