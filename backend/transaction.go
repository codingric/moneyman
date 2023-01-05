package main

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

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
	started := time.Now()
	defer func() { log.Trace().Caller().Dur("duration_ms", time.Since(started)).Send() }()

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
			log.Error().Msgf("invalid operator %s", op)
			c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid operator %s", op)})
			return
		}
	}
	var transactions []Transaction
	query.WithContext(c.Request.Context()).Find(&transactions)

	c.JSON(http.StatusOK, gin.H{"data": transactions})
}

// GET /transactions/:id
// Find a transaction
func FindTransaction(c *gin.Context) {
	started := time.Now()
	defer func() { log.Trace().Caller().Dur("duration_ms", time.Since(started)).Send() }()
	// Get model if exist
	var transaction Transaction
	if err := DB.WithContext(c.Request.Context()).Where("id = ?", c.Param("id")).First(&transaction).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Record not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": transaction})
}

// POST /transactions
// Create new transaction
func CreateTransaction(c *gin.Context) {
	started := time.Now()
	defer func() { log.Trace().Caller().Dur("duration_ms", time.Since(started)).Send() }()
	// Validate input
	var input CreateTransactionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		if Debug {
			log.Printf("CreateTransaction.error: %s", err.Error())
		}
		return
	}

	b := []string{fmt.Sprint(input.Created.Unix()), input.Account, input.Amount, input.Description}
	d := strings.Join(b, "!")
	h := md5.Sum([]byte(d))
	if Debug {
		log.Printf("Hash: %x", h)
	}

	// Create transaction
	f, _ := strconv.ParseFloat(input.Amount, 64)
	a, _ := strconv.ParseInt(input.Account, 10, 64)
	//t, _ := time.Parse("2006-01-02", input.Created)
	transaction := Transaction{Md5: fmt.Sprintf("%x", h), Created: input.Created, Amount: f, Description: input.Description, Account: a}
	result := DB.WithContext(c.Request.Context()).Create(&transaction)

	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": result.Error})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": transaction})
}

// func Upload(c *gin.Context) {
// 	file, _, err := c.Request.FormFile("csv")
// 	if err != nil {
// 		c.String(http.StatusBadRequest, "get form err: %s", err.Error())
// 		return
// 	}

// 	reader := csv.NewReader(file)
// 	records, _ := reader.ReadAll()

// 	hashes := []string{}

// 	imported := 0
// 	skipped := []gin.H{}

// 	for index, row := range records {

// 		if row[0] == "Date" {
// 			continue
// 		}

// 		created, _ := time.Parse("02/01/2006", row[0])

// 		amount := row[3]
// 		if row[3] == "" {
// 			amount = row[4]
// 		}

// 		b := []string{fmt.Sprint(created.Unix()), row[1], amount, row[2]}
// 		d := strings.Join(b, "!")
// 		h := md5.Sum([]byte(d))

// 		hashes = append(hashes, fmt.Sprintf("%x", h))
// 		// Create transaction
// 		f, _ := strconv.ParseFloat(amount, 64)
// 		a, _ := strconv.ParseInt(row[1], 10, 64)
// 		transaction := Transaction{Md5: fmt.Sprintf("%x", h), Created: created, Amount: f, Description: row[2], Account: a}
// 		result := DB.Debug().Create(&transaction)

// 		if result.Error != nil {
// 			skipped = append(skipped, gin.H{"index": index, "error": fmt.Sprint(result.Error)})
// 			continue
// 		}
// 		imported += 1
// 	}
// 	c.JSON(http.StatusOK, gin.H{"result": gin.H{"imported": imported, "skipped": len(skipped), "errors": skipped}})
// 	return
// }
