package controllers

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"crypto/md5"
	"fmt"
	"encoding/csv"
	"path/filepath"
	"os"

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

	b := []string{input.Created.Format("2006-01-02"), input.Account, input.Amount, input.Description}
	d := strings.Join(b, "!")
	h := md5.Sum([]byte(d))

	log.Printf("Hash: %x", h)

	// Create transaction
	f, _ := strconv.ParseFloat(input.Amount, 64)
	a, _ := strconv.ParseInt(input.Account, 10, 64)
	//t, _ := time.Parse("2006-01-02", input.Created)
	transaction := models.Transaction{Md5: fmt.Sprintf("%x", h), Created: input.Created, Amount: f, Description: input.Description, Account: a}
	result := models.DB.Create(&transaction)

	if result.Error != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error":result.Error})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": transaction})
}

func Upload(c *gin.Context) {
	file, err := c.FormFile("csv")
	if err != nil {
		c.String(http.StatusBadRequest, "get form err: %s", err.Error())
		return
	}

	filename := filepath.Base(file.Filename)
	if err := c.SaveUploadedFile(file, filename); err != nil {
		c.String(http.StatusBadRequest, fmt.Sprintf("upload file err: %s", err.Error()))
		return
	}

	fp, _ := os.Open(filename)

	reader := csv.NewReader(fp)
	records, _ := reader.ReadAll()

	hashes := []string{}

	imported := 0
	skipped := []gin.H{}
	
	for index, row := range records {
		
		if row[0] == "Date" {
			continue
		}
		
		created, _ := time.Parse("02/01/2006", row[0])

		amount := row[3]
		if row[3] == "" {
			amount = row[4]
		}
	
		b := []string{created.Format("2006-01-02"), row[1], amount, row[2]}
		d := strings.Join(b, "!")
		log.Print(d)
		h := md5.Sum([]byte(d))
	
		hashes = append(hashes, fmt.Sprintf("%x", h))
		// Create transaction
		f, _ := strconv.ParseFloat(amount, 64)
		a, _ := strconv.ParseInt(row[1], 10, 64)
		transaction := models.Transaction{Md5: fmt.Sprintf("%x", h), Created: created, Amount: f, Description: row[2], Account: a}
		result := models.DB.Debug().Create(&transaction)

		if result.Error != nil {
			skipped = append(skipped, gin.H{"index":index, "error": fmt.Sprint(result.Error)})
			continue
		}
		imported += 1
	}
	c.JSON(http.StatusOK, gin.H{"result": gin.H{"imported": imported, "skipped": len(skipped), "errors": skipped}})
	return
}