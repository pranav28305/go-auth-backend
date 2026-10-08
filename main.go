package main


import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"io"
	"unicode/utf8"
	"errors"
	"log"
	"context"
	"os/signal"
	"syscall"
	"time"

	_ "modernc.org/sqlite"
)


func HomePage(w http.ResponseWriter, r *http.Request){
	fmt.Fprintln(w, "Welcome to my server")
}


func validatePerson(person Person) error {

	if person.Name == "" {
			return errors.New("Name is required")
	}

	nameLength := utf8.RuneCountInString(person.Name)

		if nameLength < 2 {
			return errors.New("Name must be atleast 2 characters")
		}

		if nameLength > 50 {
			return errors.New("Name must be at most 50 characters")
		}

		return nil
}

func HelloPage(w http.ResponseWriter, r *http.Request){
	fmt.Println("Method:", r.Method)
	fmt.Println("Path:", r.URL.Path)

	fmt.Fprintln(w, "Hello!")
}

func AboutPage(w http.ResponseWriter, r *http.Request){
	fmt.Fprintln(w, "This is the about pagge")
}

func MePage(w http.ResponseWriter, r *http.Request){
	if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		name := r.URL.Query().Get("name")

		if name == "" {
			fmt.Fprintln(w, "Please provide a name for yourself")
			return
		} 

		user := Person{
			Name: name,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
}

func PersonPage(db *sql.DB) http.HandlerFunc { return func(w http.ResponseWriter, r *http.Request) {
		
		contentType := r.Header.Get("Content-Type")

		if contentType != "application/json" {
			http.Error(w, "Content-type must be application/json", http.StatusUnsupportedMediaType)
			return
		}

		var person Person

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) 
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&person)

		if err != nil {
			fmt.Println("Decode error:", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return 
		}

		err = decoder.Decode(&struct{}{})

		if err != io.EOF{
			http.Error(w, "Request body must contain only one JSON object", http.StatusBadRequest)
			return
		}

		err = validatePerson(person)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		person, err = createPerson(db, person)

		if err != nil{
			log.Println("Database error:", err)
			http.Error(w, "Failed to create person", http.StatusInternalServerError)
			return
		}

		response  := PersonResponse{
			Message: "Person created",
			Person: person,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
}}

func PeoplePage(db *sql.DB) http.HandlerFunc { return func(w http.ResponseWriter, r *http.Request) {

	people, err := getPeople(db)

	if err != nil {
		log.Println("Database error:", err)
		http.Error(w, "Failed to get people", http.StatusInternalServerError)
		return
	}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(people)
}}

/*func PersonGetSet(w http.ResponseWriter, r *http.Request) {
	
		idString := r.PathValue("id")
		id, err := strconv.Atoi(idString)

		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		switch r.Method {

		case http.MethodGet:

			for _,person := range people {

				if person.ID == id {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(person)
					return
				}
			}
			
			http.Error(w, "Person not found", http.StatusNotFound)

		case http.MethodDelete:

			for i, person := range people {
				if person.ID == id {

					people = append(people[:i], people[i+1]...)

					w.Header().Set("Content-Type", "application/json")
					fmt.Fprintln(w, "Person deleted")
					return
				}
			}

			http.Error(w, "Person not found", http.StatusNotFound)

		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
		
}*/

func PersonGet(db *sql.DB) http.HandlerFunc { return func(w http.ResponseWriter, r *http.Request) {
	idString := r.PathValue("id")
		id, err := strconv.Atoi(idString)

		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

	person, err := getPerson(db, id) 

	if err != nil {

		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "Person Not Found", http.StatusNotFound)
			return
		}

		log.Println("Database error:", err)
		http.Error(w, "Failed to get person", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(person)

}}

func PersonDelete(db *sql.DB) http.HandlerFunc { return func(w http.ResponseWriter, r *http.Request){
	idString := r.PathValue("id")
		id, err := strconv.Atoi(idString)

		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

	deleted, err := deletePerson(db, id)

	if err != nil {
		log.Println("Database error:", err)
		http.Error(w, "Failed to delete person", http.StatusInternalServerError)
		return
	}

	if !deleted {
		http.Error(w, "Person Not Found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	fmt.Fprintln(w,`{"message":"Person deleted"}`)
}}


func RegisterPage(db *sql.DB) http.HandlerFunc { return func(w http.ResponseWriter, r *http.Request) {
	var request RegisterRequest

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	err := decoder.Decode(&request)

	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if request.Username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}

	if request.Password == "" {
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}

	passwordHash, err := hashPassword(request.Password)

	if err != nil {
		log.Println("Password hashing error:", err)
		http.Error(w, "Failed to process password", http.StatusInternalServerError)
		return
	}

	user := User{
		Username: request.Username,
		PasswordHash : passwordHash,
	}

	user, err = createUser(db, user)

	if err != nil {
		
		if errors.Is(err, ErrUsernameTaken) {
			http.Error(w, "Username already exists", http.StatusConflict)
			return
		}

		log.Println("Database error:", err)

		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	json.NewEncoder(w).Encode(user)

}}


type Person struct {
	ID int `json:"id"`
	Name string `json:"name"`
}


type PersonResponse struct {
	Message string `json:"message"`
	Person Person `json:"person"`
}

type User struct {
	ID int `json:"id"`
	Username string `json:"username"`
	PasswordHash string `json:"-"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	User 	User `json:"user"`
	CSRFToken string `json:"csrf_token"`
}


func main() {

//	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
//		fmt.Fprintln(w, "Welcome to my server")
//	})		

	db := initDB()
	defer db.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("/", HomePage)



	mux.HandleFunc("/hello", HelloPage)



	mux.HandleFunc("/about", AboutPage)



	mux.HandleFunc("/me", MePage)



	mux.Handle("POST /person", authMiddleware(csrfMiddleware(PersonPage(db),),),)



	mux.HandleFunc("GET /people", PeoplePage(db))



	mux.HandleFunc("GET /person/{id}", PersonGet(db))

	mux.Handle("DELETE /person/{id}", authMiddleware(csrfMiddleware(PersonDelete(db),),),)

	mux.HandleFunc("POST /register", RegisterPage(db))

	mux.Handle("POST /login", LoginRateLimit(http.HandlerFunc(LoginPage(db)),),)

	mux.Handle("GET /profile", authMiddleware(ProfilePage(db),),)

	mux.Handle("POST /logout", authMiddleware(csrfMiddleware(http.HandlerFunc(LogoutPage),),),)
	
	server := &http.Server {
		Addr : ":8080",
		Handler : loggingMiddleware(mux),
	}

	go func() {
			log.Println("Server running on http://localhost:8080")

			err := server.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				log.Fatal(err)
			}
	}()

	ctx, stop := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT,
		syscall.SIGTERM,
	)
	defer stop()

	go startSessionCleanup(ctx)

	<-ctx.Done()
	log.Println("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Println("Server shutdown error:", err)
	}

	log.Println("Server stopped")
}


/*
package main


import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"io"
	"unicode/utf8"
	"errors"
	"log"
)


func HomePage(w http.ResponseWriter, r *http.Request){
	fmt.Fprintln(w, "Welcome to my server")
}


func validatePerson(person Person) error {

	if person.Name == "" {
			return errors.New("Name is required")
	}

	nameLength := utf8.RuneCountInString(person.Name)

		if nameLength < 2 {
			return errors.New("Name must be atleast 2 characters")
		}

		if nameLength > 50 {
			return errors.New("Name must be at most 50 characters")
		}

		return nil
}


var nextId int = 1


type Person struct {
	ID int `json:"id"`
	Name string `json:"name"`
}


type PersonResponse struct {
	Message string `json:"message"`
	Person Person `json:"person"`
}


var people []Person //to return the http response in the get method


func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

	fmt.Println("Method:", r.Method)
	fmt.Println("Path:", r.URL.Path)

	next.ServeHTTP(w, r)
	})
}


func main() {

//	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
//		fmt.Fprintln(w, "Welcome to my server")
//	})		

	http.HandleFunc("/", HomePage)



	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Method:", r.Method)
		fmt.Println("Path:", r.URL.Path)

		fmt.Fprintln(w, "Hello!")
	})



	http.HandleFunc("/about", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "This is the about page.")
	})



	http.HandleFunc("/me", func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		name := r.URL.Query().Get("name")

		if name == "" {
			fmt.Fprintln(w, "Please provide a name for yourself")
			return
		} 

		user := Person{
			Name: name,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	})



	http.HandleFunc("/person", func(w http.ResponseWriter, r *http.Request) {
		
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return 
		}

		contentType := r.Header.Get("Content-Type")

		if contentType != "application/json" {
			http.Error(w, "Content-type must be application/json", http.StatusUnsupportedMediaType)
			return
		}

		var person Person

		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&person)

		if err != nil {
			fmt.Println("Decode error:", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return 
		}

		err = decoder.Decode(&struct{}{})

		if err != io.EOF{
			http.Error(w, "Request body must contain only one JSON object", http.StatusBadRequest)
			return
		}

		err = validatePerson(person)

		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		person.ID = nextId
		nextId++

		people = append(people, person)

		response  := PersonResponse{
			Message: "Person created",
			Person: person,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(response)
	})





	http.HandleFunc("/people", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(people)
	})



	http.HandleFunc("/person/{id}", func(w http.ResponseWriter, r *http.Request){

		idString := r.PathValue("id")
		id, err := strconv.Atoi(idString)

		if err != nil {
			http.Error(w, "Invalid ID", http.StatusBadRequest)
			return
		}

		switch r.Method {

		case http.MethodGet:

			for _,person := range people {

				if person.ID == id {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(person)
					return
				}
			}
			
			http.Error(w, "Person not found", http.StatusNotFound)

		case http.MethodDelete:

			for i, person := range people {
				if person.ID == id {

					people = append(people[:i], people[i+1]...)

					w.Header().Set("Content-Type", "application/json")
					fmt.Fprintln(w, "Person deleted")
					return
				}
			}

			http.Error(w, "Person not found", http.StatusNotFound)

		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
		
	})





	log.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
	*/