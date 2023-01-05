package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"

	"github.com/gin-gonic/gin"
)

type CreateTransactionInput struct {
	Created     time.Time `json:"created" binding:"required"`
	Amount      string    `json:"amount" binding:"required"`
	Description string    `json:"description" binding:"required"`
	Account     string    `json:"account" binding:"required"`
}

type Transaction struct {
	ID          uint      `json:"id" gorm:"primary_key"`
	Md5         string    `json:"-" gorm:"unique"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	Account     int64     `json:"account"`
	Created     time.Time `json:"created"`
}

// GET /transactions
// Find all transactions
func FindTransactions(c *gin.Context) {
	ctx := context.Background()
	if c.Request != nil {
		ctx = c.Request.Context()
	}

	ctx, span := otel.Tracer("").Start(ctx, "transaction.FindTransactions")
	defer span.End()

	filters := c.Request.URL.Query()
	query := DB
	for filter, value := range filters {
		p := strings.Split(filter, "__")
		op := "eq"
		field := filter
		val := value[0]

		if len(p) > 1 {
			field = p[0]
			op = p[1]
		}
		switch field {
		case "created":
			d, _ := time.Parse("2006-01-02T15:04:05", val)
			val = d.Format("2006-01-02 15:04:05")
		}
		log.Debug().Msgf("Filter: %v %v %v", field, op, val)

		switch op {
		case "like":
			query = query.Where(field+" LIKE ?", "%"+val+"%")
		case "gt":
			query = query.Where(field+" > ?", val)
		case "ge":
			query = query.Where(field+" >= ?", val)
		case "lt":
			query = query.Where(field+" < ?", val)
		case "le":
			query = query.Where(field+" <= ?", val)
		case "eq":
			query = query.Where(field+" = ?", val)
		case "ne":
			query = query.Where(field+" != ?", val)
		default:
			msg := fmt.Sprintf("invalid operator %s", op)
			log.Error().Msg(msg)
			c.JSON(http.StatusBadRequest, gin.H{"error": msg})
			span.SetStatus(codes.Error, "msg")
			return
		}
	}
	var transactions []Transaction
	if err := query.WithContext(ctx).Find(&transactions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Unable to retreive data"})
		//span.RecordError(err)
		span.SetStatus(codes.Error, "Unable to retreive data")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": transactions})
}

// GET /transactions/:id
// Find a transaction
func FindTransaction(c *gin.Context) {
	ctx := context.Background()
	if c.Request != nil {
		ctx = c.Request.Context()
	}

	ctx, span := otel.Tracer("").Start(ctx, "transaction.FindTransaction")
	defer span.End()

	// Get model if exist
	var transaction Transaction
	if err := DB.WithContext(ctx).Where("id = ?", c.Param("id")).First(&transaction).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		span.RecordError(err)
		span.SetStatus(codes.Error, "Record not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": transaction})
}

// POST /transactions
// Create new transaction
func CreateTransaction(c *gin.Context) {
	ctx := context.Background()
	if c.Request != nil {
		ctx = c.Request.Context()
	}

	ctx, span := otel.Tracer("").Start(ctx, "transaction.CreateTransaction")
	defer span.End()

	// Validate input
	var input CreateTransactionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Error().Caller().Err(err)
		return
	}

	b := []string{fmt.Sprint(input.Created.Unix()), input.Account, input.Amount, input.Description}
	d := strings.Join(b, "!")
	h := md5.Sum([]byte(d))
	log.Debug().Caller().Msgf("Hash: %x", h)

	// Create transaction
	f, _ := strconv.ParseFloat(input.Amount, 64)
	a, _ := strconv.ParseInt(input.Account, 10, 64)
	//t, _ := time.Parse("2006-01-02", input.Created)
	transaction := Transaction{Md5: fmt.Sprintf("%x", h), Created: input.Created, Amount: f, Description: input.Description, Account: a}
	result := DB.WithContext(ctx).Create(&transaction)

	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": transaction})
}
