package controllers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/codingric/moneyman/backend/models"
	"github.com/gin-gonic/gin"
)

type CreateTransactionInput struct {
	Created     time.Time `json:"created" binding:"required"`
	Amount      string    `json:"amount" binding:"required"`
	Description string    `json:"description" binding:"required"`
	Account     string    `json:"account" binding:"required"`
}

// GET /transactions
// Find all transactions
func FindTransactions(c *gin.Context) {
	filters := c.Request.URL.Query()
	query := models.DB
	for filter, value := range filters {
		p := strings.Split(filter, "__")
		op := "eq"
		field := filter
		if len(p) > 1 {
			field = p[0]
			op = p[1]
		}
		log.Printf("Filter: %v %v %v", field, op, value[0])
		switch op {
		case "like":
			query = query.Where(field+" LIKE ?", "%"+value[0]+"%")
		case "gt":
			query = query.Where(field+" > ?", value[0])
		case "ge":
			query = query.Where(field+" >= ?", value[0])
		case "lt":
			query = query.Where(field+" < ?", value[0])
		case "le":
			query = query.Where(field+" <= ?", value[0])
		default:
			query = query.Where(field+" = ?", value[0])
		}
	}
	var transactions []models.Transaction
	query.Find(&transactions)

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
	a, _ := strconv.ParseInt(input.Account, 10, 64)
	//t, _ := time.Parse("2006-01-02", input.Created)
	transaction := models.Transaction{Created: input.Created, Amount: f, Description: input.Description, Account: a}
	models.DB.Create(&transaction)

	c.JSON(http.StatusOK, gin.H{"data": transaction})
}
