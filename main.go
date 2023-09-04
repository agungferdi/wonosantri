package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/wcharczuk/go-chart"
)

type StockItem struct {
	ID          int
	StockID     int
	Dates       time.Time
	CoffeeType  string
	Pack1       int
	Pack2       int
	Pack3       int
	Amount1     int
	Amount2     int
	Amount3     int
	TotalAmount int
}

var tmpl = template.Must(template.ParseGlob("templates/*.html"))

func main() {
	db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/wonosantri")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT stockid, dates, coffeetype, pack1, pack2, pack3, amount1, amount2, amount3, totalamount FROM wonosantristock")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		var stockItems []StockItem
		for rows.Next() {
			var stock StockItem
			var dateString string
			err := rows.Scan(
				&stock.ID,
				&dateString,
				&stock.CoffeeType,
				&stock.Pack1,
				&stock.Pack2,
				&stock.Pack3,
				&stock.Amount1,
				&stock.Amount2,
				&stock.Amount3,
				&stock.TotalAmount,
			)
			if err != nil {
				log.Fatal(err)
			}

			stock.Dates, err = time.Parse("2006-01-02 15:04:05", dateString)
			if err != nil {
				log.Fatal(err)
			}

			stockItems = append(stockItems, stock)
		}

		err = tmpl.ExecuteTemplate(w, "index.html", stockItems)
		if err != nil {
			log.Fatal(err)
		}
	})

	http.HandleFunc("/create", createHandler)
	http.HandleFunc("/update", updateHandler)
	http.HandleFunc("/delete", deleteHandler)
	http.HandleFunc("/summary", summaryHandler)

	fs := http.FileServer(http.Dir("assets"))
	http.Handle("/assets/", http.StripPrefix("/assets/", fs))

	fmt.Println("Server started at :7000")
	http.ListenAndServe(":7000", nil)

}

func createHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		dates := r.FormValue("dates")
		coffeetype := r.FormValue("coffeetype")
		pack1, _ := strconv.Atoi(r.FormValue("pack1"))
		pack2, _ := strconv.Atoi(r.FormValue("pack2"))
		pack3, _ := strconv.Atoi(r.FormValue("pack3"))

		// Calculate temporary Amount1, Amount2, Amount3, and TotalAmount for review
		amount1 := pack1 * 200
		amount2 := pack2 * 500
		amount3 := pack3 * 1000
		totalamount := amount1 + amount2 + amount3

		if r.FormValue("final") == "true" {
			db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/wonosantri")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer db.Close()

			// Insert the data into the database
			insertQuery := "INSERT INTO wonosantristock (dates, coffeetype, pack1, pack2, pack3, amount1, amount2, amount3, totalamount) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)"
			_, err = db.Exec(insertQuery, dates, coffeetype, pack1, pack2, pack3, amount1, amount2, amount3, totalamount)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, "/", http.StatusSeeOther)
		} else {
			reviewData := struct {
				Dates       string
				CoffeeType  string
				Pack1       int
				Pack2       int
				Pack3       int
				Amount1     int
				Amount2     int
				Amount3     int
				TotalAmount int
			}{
				Dates:       dates,
				CoffeeType:  coffeetype,
				Pack1:       pack1,
				Pack2:       pack2,
				Pack3:       pack3,
				Amount1:     amount1,
				Amount2:     amount2,
				Amount3:     amount3,
				TotalAmount: totalamount,
			}

			data := struct {
				Review interface{}
			}{
				Review: reviewData,
			}

			tmpl.ExecuteTemplate(w, "create.html", data)
		}
	} else {
		tmpl.ExecuteTemplate(w, "create.html", nil)
	}
}

func updateHandler(w http.ResponseWriter, r *http.Request) {
	// Check if the request method is POST (form submission)
	if r.Method == http.MethodPost {
		// Parse the form data
		err := r.ParseForm()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Retrieve the form values
		stockID, _ := strconv.Atoi(r.FormValue("stockid"))
		dates := r.FormValue("dates")
		coffeetype := r.FormValue("coffeetype")
		pack1, _ := strconv.Atoi(r.FormValue("pack1"))
		pack2, _ := strconv.Atoi(r.FormValue("pack2"))
		pack3, _ := strconv.Atoi(r.FormValue("pack3"))
		// Add lines to retrieve the additional fields
		amount1, _ := strconv.Atoi(r.FormValue("amount1"))
		amount2, _ := strconv.Atoi(r.FormValue("amount2"))
		amount3, _ := strconv.Atoi(r.FormValue("amount3"))
		totalamount, _ := strconv.Atoi(r.FormValue("totalamount"))

		// Update the stock item in the database
		db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/wonosantri")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer db.Close()

		updateQuery := "UPDATE wonosantristock SET dates=?, coffeetype=?, pack1=?, pack2=?, pack3=?, amount1=?, amount2=?, amount3=?, totalamount=? WHERE stockid=?"
		_, err = db.Exec(updateQuery, dates, coffeetype, pack1, pack2, pack3, amount1, amount2, amount3, totalamount, stockID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Redirect the user back to the main page
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// If the request method is not POST (e.g., GET), retrieve the stockid from the URL parameters
	stockID, err := strconv.Atoi(r.URL.Query().Get("stockid"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Retrieve the existing data for the selected stock item based on stockID
	db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/wonosantri")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer db.Close()

	row := db.QueryRow("SELECT dates, coffeetype, pack1, pack2, pack3, amount1, amount2, amount3, totalamount FROM wonosantristock WHERE stockid=?", stockID)

	var dates, coffeetype string
	var pack1, pack2, pack3, amount1, amount2, amount3, totalamount int

	err = row.Scan(&dates, &coffeetype, &pack1, &pack2, &pack3, &amount1, &amount2, &amount3, &totalamount)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a struct to store the retrieved data
	stockItem := struct {
		StockID     int
		Dates       string
		CoffeeType  string
		Pack1       int
		Pack2       int
		Pack3       int
		Amount1     int
		Amount2     int
		Amount3     int
		TotalAmount int
	}{
		StockID:     stockID,
		Dates:       dates,
		CoffeeType:  coffeetype,
		Pack1:       pack1,
		Pack2:       pack2,
		Pack3:       pack3,
		Amount1:     amount1,
		Amount2:     amount2,
		Amount3:     amount3,
		TotalAmount: totalamount,
	}

	// Execute the update template and pass the data
	tmpl.ExecuteTemplate(w, "update.html", stockItem)
}

func deleteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// Parse form values
		r.ParseForm()
		stockID := r.FormValue("stockid")

		// Convert form values to appropriate data types
		stockIDInt, err := strconv.Atoi(stockID)
		if err != nil {
			log.Fatal(err)
		}

		// Perform DELETE operation using the converted value
		db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/wonosantri")
		if err != nil {
			log.Fatal(err)
		}
		defer db.Close()

		deleteQuery := "DELETE FROM wonosantristock WHERE stockid = ?"
		_, err = db.Exec(deleteQuery, stockIDInt)
		if err != nil {
			log.Fatal(err)
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	tmpl.ExecuteTemplate(w, "delete.html", nil)
}
func summaryHandler(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("mysql", "root:@tcp(localhost:3306)/wonosantri")
	if err != nil {
		log.Fatal(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	// Calculate the total amounts of Arabica and Robusta coffee
	var arabicaTotal int
	var robustaTotal int

	// Query your database to get the total amounts
	query := `
        SELECT 
            SUM(COALESCE(CASE WHEN coffeetype = 'Arabica' THEN totalamount ELSE 0 END, 0)) AS arabica_total,
            SUM(COALESCE(CASE WHEN coffeetype = 'Robusta' THEN totalamount ELSE 0 END, 0)) AS robusta_total
        FROM wonosantristock
        WHERE coffeetype IN ('Arabica', 'Robusta');
    `

	err = db.QueryRow(query).Scan(&arabicaTotal, &robustaTotal)
	if err != nil {
		log.Fatal(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Pass these values to the template
	data := struct {
		ArabicaTotal int
		RobustaTotal int
	}{
		ArabicaTotal: arabicaTotal,
		RobustaTotal: robustaTotal,
	}

	// Render the summary.html template with the data
	tmpl.ExecuteTemplate(w, "summary.html", data)
}
