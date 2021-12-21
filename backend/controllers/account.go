package controllers

import (
	"net/http"
	"strconv"

	"models"

	"github.com/gin-gonic/gin"
)

type CreateAccountInput struct {
	ID   string `json:"id" binding:"required"`
	Name string `json:"name" binding:"required"`
}

type UpdateAccountInput struct {
	ID   string `json:"id"`
	Name string `json:"author"`
}

// GET /accounts
// Find all accounts
func FindAccounts(c *gin.Context) {
	var accounts []models.Account
	models.DB.Find(&accounts)

	c.JSON(http.StatusOK, gin.H{"data": accounts})
}

// GET /accounts/:id
// Find a account
func FindAccount(c *gin.Context) {
	// Get model if exist
	var account models.Account
	if err := models.DB.Where("id = ?", c.Param("id")).First(&account).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Record not found!"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": account})
}

// POST /accounts
// Create new account
func CreateAccount(c *gin.Context) {
	// Validate input
	var input CreateAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create account
	i, _ := strconv.ParseInt(input.ID, 10, 64)
	account := models.Account{ID: i, Name: input.Name}
	models.DB.Create(&account)

	c.JSON(http.StatusOK, gin.H{"data": account})
}

// PATCH /accounts/:id
// Update a account
func UpdateAccount(c *gin.Context) {
	// Get model if exist
	var account models.Account
	if err := models.DB.Where("id = ?", c.Param("id")).First(&account).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Record not found!"})
		return
	}

	// Validate input
	var input UpdateAccountInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	models.DB.Model(&account).Updates(input)

	c.JSON(http.StatusOK, gin.H{"data": account})
}

// DELETE /accounts/:id
// Delete a account
func DeleteAccount(c *gin.Context) {
	// Get model if exist
	var account models.Account
	if err := models.DB.Where("id = ?", c.Param("id")).First(&account).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Record not found!"})
		return
	}

	models.DB.Delete(&account)

	c.JSON(http.StatusOK, gin.H{"data": true})
}
