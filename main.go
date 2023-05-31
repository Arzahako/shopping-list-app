package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
)

// Database variable
var db *sql.DB

// Struct to hold flash messages
type FlashMessage struct {
	Message string
}

type Product struct {
	Product  string
	Quantity int
	Store    string
}

type ListData struct {
	ListName string
	Products []Product
}

func main() {
	// Establish a database connection
	var err error
	db, err = sql.Open("mysql", "root:w8-!oY4-taa630-lsKnW0ut@tcp(localhost:3306)/shopping_list_app")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Serve static files from the "static" directory
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Register routes
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/register", registerHandler)
	http.HandleFunc("/register-success", registerSuccessHandler)
	http.HandleFunc("/login-success", loginSuccessHandler)
	http.HandleFunc("/create-list", createListHandler)
	http.HandleFunc("/list-success", listSuccessHandler)
	http.HandleFunc("/view-lists", viewListsHandler)

	// Start the server
	log.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderTemplate(w, "index.html", nil)
	}
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		email := r.PostForm.Get("email")
		password := r.PostForm.Get("password")

		// Check the credentials in the database
		username, err := checkCredentials(db, email, password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if username == "" {
			data := struct {
				ErrorMessage string
			}{
				ErrorMessage: "Wrong credentials",
			}
			renderTemplate(w, "login.html", data)
			return
		}

		data := struct {
			Username string
		}{
			Username: username,
		}

		renderTemplate(w, "login-success.html", data)
		return
	}

	renderTemplate(w, "login.html", nil)
}

func checkCredentials(db *sql.DB, email, password string) (string, error) {
	query := "SELECT name FROM users WHERE email = ? AND password = ?"
	var username string
	err := db.QueryRow(query, email, password).Scan(&username)
	if err != nil {
		return "", err
	}
	return username, nil
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		name := r.PostForm.Get("name")
		email := r.PostForm.Get("email")
		password := r.PostForm.Get("password")

		if name == "" || email == "" || password == "" {
			http.Error(w, "All fields are required", http.StatusBadRequest)
			return
		}

		if !isValidEmail(email) {
			http.Error(w, "Invalid email", http.StatusBadRequest)
			return
		}

		if len(password) < 8 {
			data := struct {
				ErrorMessage string
			}{
				ErrorMessage: "Password should be at least 8 characters long",
			}
			renderTemplate(w, "register.html", data)
			return
		}

		// Check if the email already exists in the database
		emailExists, err := isEmailExists(db, email)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if emailExists {
			data := struct {
				ErrorMessage string
			}{
				ErrorMessage: "Email already exists",
			}
			renderTemplate(w, "register.html", data)
			return
		}

		// Insert the new user into the database
		insertQuery := "INSERT INTO users (name, email, password) VALUES (?, ?, ?)"
		_, err = db.Exec(insertQuery, name, email, password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Redirect to the register success page
		http.Redirect(w, r, "/register-success", http.StatusFound)
		return
	}

	renderTemplate(w, "register.html", nil)
}

func loginSuccessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderTemplate(w, "login-success.html", nil)
	}
}

func registerSuccessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		renderTemplate(w, "register-success.html", nil)
	}
}

func renderTemplate(w http.ResponseWriter, tmpl string, data interface{}) {
	tmpl = fmt.Sprintf("templates/%s", tmpl)
	t, err := template.ParseFiles(tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func isValidEmail(email string) bool {
	// Email validation regex pattern
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	match, _ := regexp.MatchString(pattern, email)
	return match
}

func isEmailExists(db *sql.DB, email string) (bool, error) {
	query := "SELECT COUNT(*) FROM users WHERE email = ?"
	var count int
	err := db.QueryRow(query, email).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func validateCredentials(db *sql.DB, email, password string) (bool, error) {
	query := "SELECT COUNT(*) FROM users WHERE email = ? AND password = ?"
	var count int
	err := db.QueryRow(query, email, password).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func createListHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		listName := r.PostForm.Get("listName")

		// Check if the list name already exists in the database
		listExists, err := isListExists(db, listName)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if listExists {
			data := struct {
				ErrorMessage string
			}{
				ErrorMessage: "The list with such name already exists",
			}
			renderTemplate(w, "create-list.html", data)
			return
		}

		// Insert the new list into the database
		insertListQuery := "INSERT INTO lists (user_id, name) VALUES (?, ?)"
		res, err := db.Exec(insertListQuery, 1, listName) // Replace 1 with the appropriate user ID
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		listID, _ := res.LastInsertId()

		// Get the product, quantity, and store values from the form
		products := r.PostForm["product[]"]
		quantities := r.PostForm["quantity[]"]
		stores := r.PostForm["store[]"]

		// Insert each product into the database
		insertProductQuery := "INSERT INTO products (list_id, name, quantity, store) VALUES (?, ?, ?, ?)"
		for i := range products {
			_, err = db.Exec(insertProductQuery, listID, products[i], quantities[i], stores[i])
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// Redirect to the list success page
		http.Redirect(w, r, fmt.Sprintf("/list-success?name=%s", listName), http.StatusFound)
		return
	}

	renderTemplate(w, "create-list.html", nil)
}

func isListExists(db *sql.DB, listName string) (bool, error) {
	query := "SELECT COUNT(*) FROM lists WHERE name = ?"
	var count int
	err := db.QueryRow(query, listName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func listSuccessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		listName := r.URL.Query().Get("name")
		data := struct {
			ListName string
		}{
			ListName: listName,
		}
		renderTemplate(w, "list-success.html", data)
	}
}

func viewListsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		// Retrieve the list names and items from the database
		lists, err := getListsData(db)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		data := struct {
			Lists []ListData
		}{
			Lists: lists,
		}

		renderTemplate(w, "view-lists.html", data)
	}
}

func getListsData(db *sql.DB) ([]ListData, error) {
	query := "SELECT id, name FROM lists"
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lists []ListData

	for rows.Next() {
		var listID int
		var listName string

		err := rows.Scan(&listID, &listName)
		if err != nil {
			return nil, err
		}

		products, err := getProductsData(db, listID)
		if err != nil {
			return nil, err
		}

		listData := ListData{
			ListName: listName,
			Products: products,
		}

		lists = append(lists, listData)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return lists, nil
}

func getProductsData(db *sql.DB, listID int) ([]Product, error) {
	query := "SELECT name, quantity, store FROM products WHERE list_id = ?"
	rows, err := db.Query(query, listID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []Product

	for rows.Next() {
		var name string
		var quantity int
		var store string

		err := rows.Scan(&name, &quantity, &store)
		if err != nil {
			return nil, err
		}

		product := Product{
			Product:  name,
			Quantity: quantity,
			Store:    store,
		}

		products = append(products, product)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}
