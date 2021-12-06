package controllers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/codingric/moneyman/go/backend/models"
	"github.com/gin-gonic/gin"
)

type CreateTransactionInput struct {
	Created     time.Time `json:"created" binding:"required"`
	Amount      string    `json:"amount" binding:"required"`
	Description string    `json:"description" binding:"required"`
}

// GET /transactions
// Find all transactions
func FindTransactions(c *gin.Context) {
	var transactions []models.Transaction
	models.DB.Find(&transactions)

	c.JSON(http.StatusOK, gin.H{"data": transactions})
}

// GET /transactions/:id
// Find a transaction
func FindTransaction(c *gin.Context) {
	// Get model if exist
	var transaction models.Transaction
	if err := models.DB.Where("id = ?", c.Param("id")).First(&transaction).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Record not found!"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": transaction})
}

// POST /transactions
// Create new transaction
func CreateTransaction(c *gin.Context) {
	// Validate input
	var input CreateTransactionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create transaction
	f, _ := strconv.ParseFloat(input.Amount, 64)
	transaction := models.Transaction{CreatedAt: input.Created, Amount: int64(f * 100), Description: input.Description}
	models.DB.Create(&transaction)

	c.JSON(http.StatusOK, gin.H{"data": transaction})
}
