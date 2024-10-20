package main

import (
	"database/sql"
	"log"
	"main/Dto"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

func main() {
	var err error
	dsn := "test:test@tcp(mariadb-service.mariadb.svc.cluster.local:3306)/pretest"
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	r := gin.Default()
	r.Use(cors.New(CorsConfig()))

	// Customer routes
	r.GET("/customers", getCustomers)
	r.GET("/customers/:id", getCustomer)
	r.POST("/customers", createCustomer)
	r.PUT("/customers/:id", updateCustomer)
	r.DELETE("/customers/:id", deleteCustomer)
	r.GET("/customers/:id/transactions/pastyear", getCustomerWithPastYearTransactions)

	// Transaction routes
	r.GET("/transactions", getTransactions)
	r.GET("/transactions/:id", getTransaction)
	r.POST("/transactions", createTransaction)
	r.PUT("/transactions/:id", updateTransaction)
	r.DELETE("/transactions/:id", deleteTransaction)

	r.Run(":8080")
}

func CorsConfig() cors.Config {
	corsConf := cors.DefaultConfig()
	corsConf.AllowAllOrigins = true
	corsConf.AllowMethods = []string{"GET", "POST", "DELETE", "OPTIONS", "PUT"}
	corsConf.AllowHeaders = []string{"Authorization", "Content-Type", "Upgrade", "Origin",
		"Connection", "Accept-Encoding", "Accept-Language", "Host", "Access-Control-Request-Method", "Access-Control-Request-Headers", "Access-Control-Allow-Headers", "x-requested-with"}
	return corsConf
}

// Customer handlers
func getCustomers(c *gin.Context) {
	name := c.Query("name")
	email := c.Query("email")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	offset := (page - 1) * pageSize

	var querySting string
	var queryTotalSting string
	queryTotalSting = "SELECT count(id) FROM customers WHERE 1=1 "
	querySting = "SELECT id, name, email, registration_date FROM customers WHERE 1=1"
	if len(name) > 0 {
		querySting += " AND name LIKE '%" + name + "%'"
		queryTotalSting += " AND name LIKE '%" + name + "%'"
	}
	if len(email) > 0 {
		querySting += " AND email LIKE '%" + email + "%'"
		queryTotalSting += " AND email LIKE '%" + email + "%'"
	}
	querySting += " LIMIT ? OFFSET ?"

	var totalCount int
	totalRows, err := db.Query(queryTotalSting)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	defer totalRows.Close()
	for totalRows.Next() {
		if err := totalRows.Scan(&totalCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}
	rows, err := db.Query(querySting, pageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	defer rows.Close()
	var result Dto.GetCustomersDTO
	var customers []map[string]interface{}
	for rows.Next() {
		var id int
		var name, email, registrationDate string
		if err := rows.Scan(&id, &name, &email, &registrationDate); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		customers = append(customers, gin.H{
			"id":                id,
			"name":              name,
			"email":             email,
			"registration_date": registrationDate,
		})
	}
	result.Data.Total = totalCount
	result.Data.Items = customers
	c.JSON(http.StatusOK, result)
}

func getCustomer(c *gin.Context) {
	id := c.Param("id")
	var name, email, registrationDate string
	err := db.QueryRow("SELECT name, email, registration_date FROM customers WHERE id = ?", id).Scan(&name, &email, &registrationDate)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Customer not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":                id,
		"name":              name,
		"email":             email,
		"registration_date": registrationDate,
	})
}

func createCustomer(c *gin.Context) {
	var result Dto.GetCustomersDTO
	var customer struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&customer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec("INSERT INTO customers (name, email, registration_date) VALUES (?, ?, ?)", customer.Name, customer.Email, time.Now().Format("2006-01-02"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, result)
}

func updateCustomer(c *gin.Context) {
	var resp Dto.BaseRespDTO
	id := c.Param("id")
	var customer struct {
		Id    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	bindingErr := c.BindJSON(&customer)
	if bindingErr != nil {
		resp.Code = int(9999)
		resp.Message = bindingErr.Error()
		c.JSON(http.StatusBadRequest, resp)
		c.Abort()
		return
	}
	// if err := c.ShouldBindJSON(&customer); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	// 	return
	// }

	_, err := db.Exec("UPDATE customers SET name = ?, email = ? WHERE id = ?", customer.Name, customer.Email, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func deleteCustomer(c *gin.Context) {
	var resp Dto.BaseRespDTO
	id := c.Param("id")
	_, err := db.Exec("DELETE FROM transactions WHERE customer_id = ?", id)
	if err != nil {
		resp.Code = int(9999)
		resp.Message = err.Error()
		c.JSON(http.StatusBadRequest, resp)
		c.Abort()
		return
	}
	_, customerErr := db.Exec("DELETE FROM customers WHERE id = ?", id)
	if customerErr != nil {
		resp.Code = int(9999)
		resp.Message = customerErr.Error()
		c.JSON(http.StatusBadRequest, resp)
		c.Abort()
		return
	}
	c.JSON(http.StatusOK, resp)
}

// Transaction handlers
func getTransactions(c *gin.Context) {

	name := c.Query("name")
	email := c.Query("email")
	startDate := c.DefaultQuery("startDate", "Mon, 01 Jan 2024 00:00:00 GMT")
	endDate := c.DefaultQuery("endDate", "Tue, 31 Dec 2024 00:00:00 GMT")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	startParsedTime, err := time.Parse(time.RFC1123, startDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	startFormattedDate := startParsedTime.Format("2006-01-02 15:04:05")

	endParsedTime, err := time.Parse(time.RFC1123, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	endFormattedDate := endParsedTime.Format("2006-01-02 15:04:05")

	offset := (page - 1) * pageSize

	queryTotalSting := `SELECT count(c.id) FROM transactions t 
	JOIN customers c on t.customer_id = c.id 
	WHERE 1=1 `

	querySting := `SELECT  name, transaction_date, amount, transaction_number
	FROM transactions t 
	JOIN customers c on t.customer_id = c.id
	WHERE 1=1 `

	if len(name) > 0 {
		querySting += " AND c.name LIKE '%" + name + "%'"
		queryTotalSting += " AND c.name LIKE '%" + name + "%'"
	}
	if len(email) > 0 {
		querySting += " AND c.email LIKE '%" + email + "%'"
		queryTotalSting += " AND c.email LIKE '%" + email + "%'"
	}
	if len(startDate) > 0 {
		querySting += " AND t.transaction_date >= '" + startFormattedDate + "'"
		queryTotalSting += " AND t.transaction_date >= '" + startFormattedDate + "'"
	}
	if len(endDate) > 0 {
		querySting += " AND t.transaction_date <= '" + endFormattedDate + "'"
		queryTotalSting += " AND t.transaction_date <= '" + endFormattedDate + "'"
	}

	totalRows, err := db.Query(queryTotalSting)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer totalRows.Close()
	var totalCount int
	for totalRows.Next() {
		if err := totalRows.Scan(&totalCount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	querySting += " LIMIT ? OFFSET ?"
	rows, err := db.Query(querySting, pageSize, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var result Dto.GetCustomersDTO

	var transactions []map[string]interface{}
	for rows.Next() {
		var amount int
		var name, transactionDate, transactionNumber string
		if err := rows.Scan(&name, &transactionDate, &amount, &transactionNumber); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		transactions = append(transactions, gin.H{
			"name":               name,
			"transaction_date":   transactionDate,
			"amount":             amount,
			"transaction_number": transactionNumber,
		})
	}
	result.Data.Total = totalCount
	result.Data.Items = transactions
	c.JSON(http.StatusOK, result)
}

func getTransaction(c *gin.Context) {
	id := c.Query("id")
	var customerID, amount, transactionNumber int
	var transactionDate string
	err := db.QueryRow("SELECT customer_id, transaction_date, amount, transaction_number FROM transactions WHERE id = ?", id).Scan(&customerID, &transactionDate, &amount, &transactionNumber)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "Transaction not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":                 id,
		"customer_id":        customerID,
		"transaction_date":   transactionDate,
		"amount":             amount,
		"transaction_number": transactionNumber,
	})
}

func createTransaction(c *gin.Context) {
	var transaction struct {
		CustomerID        int    `json:"customer_id"`
		TransactionDate   string `json:"transaction_date"`
		Amount            int    `json:"amount"`
		TransactionNumber int    `json:"transaction_number"`
	}
	if err := c.ShouldBindJSON(&transaction); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := db.Exec("INSERT INTO transactions (customer_id, transaction_date, amount, transaction_number) VALUES (?, ?, ?, ?)", transaction.CustomerID, transaction.TransactionDate, transaction.Amount, transaction.TransactionNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	id, _ := result.LastInsertId()
	c.JSON(http.StatusCreated, gin.H{"id": id})
}

func updateTransaction(c *gin.Context) {
	id := c.Param("id")
	var transaction struct {
		CustomerID        int    `json:"customer_id"`
		TransactionDate   string `json:"transaction_date"`
		Amount            int    `json:"amount"`
		TransactionNumber int    `json:"transaction_number"`
	}
	if err := c.ShouldBindJSON(&transaction); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := db.Exec("UPDATE transactions SET customer_id = ?, transaction_date = ?, amount = ?, transaction_number = ? WHERE id = ?", transaction.CustomerID, transaction.TransactionDate, transaction.Amount, transaction.TransactionNumber, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "Transaction updated"})
}

func deleteTransaction(c *gin.Context) {
	id := c.Param("id")
	_, err := db.Exec("DELETE FROM transactions WHERE id = ?", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "Transaction deleted"})
}

func getCustomerWithPastYearTransactions(c *gin.Context) {
	id := c.Param("id")
	var result Dto.GetCustomersDTO

	oneYearAgo := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	rows, err := db.Query("SELECT id, transaction_date, amount, transaction_number FROM transactions WHERE customer_id = ? AND transaction_date >= ?", id, oneYearAgo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var transactions []map[string]interface{}
	for rows.Next() {
		var transactionID, amount, transactionNumber int
		var transactionDate string
		if err := rows.Scan(&transactionID, &transactionDate, &amount, &transactionNumber); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		transactions = append(transactions, gin.H{
			"id":                 transactionID,
			"transaction_date":   transactionDate,
			"amount":             amount,
			"transaction_number": transactionNumber,
		})
	}
	result.Data.Items = transactions
	c.JSON(http.StatusOK, result)
}
