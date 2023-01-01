package main

import (
	"encoding/gob"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Gideon-isa/booking/internal/config"
	"github.com/Gideon-isa/booking/internal/driver"
	"github.com/Gideon-isa/booking/internal/handlers"
	"github.com/Gideon-isa/booking/internal/helpers"
	"github.com/Gideon-isa/booking/internal/models"
	"github.com/Gideon-isa/booking/internal/render"
	"github.com/alexedwards/scs/v2"
	"github.com/joho/godotenv"
)

const portNumber string = ":8080"

var app config.AppConfig

var session *scs.SessionManager
var infoLog *log.Logger
var errorLog *log.Logger

func main() {
	db, err := run()
	if err != nil {
		log.Fatal(err)
	}

	defer db.SQL.Close()

	defer close((app.MailChan))

	fmt.Println("Starting mail listener...")
	listenForMail()

	fmt.Printf("Starting application on port %s\n", portNumber)
	//	http.ListenAndServe(portNumber, nil)

	srv := &http.Server{
		Addr:    portNumber,
		Handler: routes(&app),
	}

	err = srv.ListenAndServe()
	log.Fatal(err)

}

func run() (*driver.DB, error) {

	// What am I doing to put in the session
	gob.Register(models.Reservation{})
	gob.Register(models.User{})
	gob.Register(models.Room{})
	gob.Register(models.Reservation{})
	gob.Register(map[string]int{})

	mailChan := make(chan models.MailData)
	app.MailChan = mailChan

	// change this to true when in production
	app.InProduction = false

	infoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	app.InfoLog = infoLog

	errorLog = log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	app.ErrorLog = errorLog

	session = scs.New()
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = app.InProduction

	app.Session = session

	// connect to databse
	//Loading the env variable using the /joho/godotenv library
	err := godotenv.Load("app.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	host := os.Getenv("HOST")
	port := os.Getenv("PORT")
	user := os.Getenv("USER")
	DBName := os.Getenv("NAME")
	password := os.Getenv("PASSWORD")

	log.Println("Connecting to database...")
	dsn := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s", host, port, DBName, user, password)
	db, err := driver.ConnectSQL(dsn)
	if err != nil {
		log.Fatal("Cannot connect to database! Dying...")
	}
	log.Println("Connected to database!")

	tc, err := render.CreateTemplateCache()
	if err != nil {
		fmt.Println(err)
		log.Fatal("cannot create template cache")

		// this is the error returned in case the run function encounters an issue
		return nil, err
	}

	app.TemplateCache = tc
	app.UseCache = false

	render.NewRenderer(&app)

	repo := handlers.NewRepo(&app, db)
	handlers.NewHandlers(repo)
	helpers.NewHelpers(&app)
	return db, nil
}
