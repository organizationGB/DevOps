package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
	"github.com/noirbizarre/gonja"
	"github.com/noirbizarre/gonja/exec"
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB

func ConnectDatabase() error {
	db, err := sql.Open("sqlite3", "./database/minitwit.db")
	if err != nil {
		return err
	}

	DB = db
	return nil
}

type User struct {
	User_id  int    `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Pw_hash  string `json:"pw_hash"`
}

type Message struct {
	Message_id     int    `json:"message_id"`
	Author_id      int    `json:"author_id"`
	Text           string `json:"text"`
	Pub_date       int    `json:"pub_date"`
	Flagged        int    `json:"flagged"`
	Formatted_date string `json:"formatted_date"`
	Username       string `json:"username"`
	Gravatar       string `json:"gravatar"`
}

func public_timeline(c *gin.Context) {
	//var ids []int

	messages := make([]Message, 0)
	users := make([]User, 0)
	session := sessions.Default(c)
	userID := session.Get("user_id")

	rows, err := DB.Query("SELECT message.*, user.* from message, user where message.flagged = 0 and message.author_id = user.user_id order by message.pub_date desc LIMIT 30")

	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()
	for rows.Next() {
		//var _id int
		msg := Message{}
		usr := User{}
		err := rows.Scan(&msg.Message_id, &msg.Author_id, &msg.Text, &msg.Pub_date, &msg.Flagged, &usr.User_id, &usr.Username, &usr.Email, &usr.Pw_hash)

		str := "Mon, 02 Jan 2006 15:04:05"
		msg.Formatted_date = time.Unix(int64(msg.Pub_date), 0).Format(str)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		msg.Username = usr.Username
		msg.Gravatar = geturl(usr.Email)
		fmt.Println(msg.Gravatar)

		messages = append(messages, msg)
		users = append(users, usr)
	}

	var tpl = gonja.Must(gonja.FromFile("template/timeline_go.html"))

	user, err := DB.Query("SELECT * FROM user WHERE username = ? LIMIT 1", userID)
	usr := User{}
	user.Scan(&usr.User_id, &usr.Username, &usr.Email, &usr.Pw_hash)

	req := struct {
		Endpoint string
	}{
		Endpoint: c.Request.URL.Path,
	}

	out, err := tpl.Execute(gonja.Context{
		"messages":     messages,
		"g":            session,
		"request":      req,
		"profile_user": usr,
	})

	fmt.Println(err)
	c.Writer.WriteString(out)
}

func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func login(c *gin.Context) {

	/*rows, err := DB.QueryRow("SELECT * FROM user WHERE username = ?", "hello")
	defer rows.Close()
	for rows.Next() {
		//var _id int
		usr := User{}
		err := rows.Scan(&usr.User_id, &usr.Username, &usr.Email, &usr.Pw_hash )
		fmt.Println(usr)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

	}*/

	session := sessions.Default(c)
	if userID := session.Get("user_id"); userID != nil {
		fmt.Println("WHAT JHJKHKJ")
		c.Redirect(http.StatusFound, "/timeline")
		return
	}

	fmt.Println(session)
	var error string
	usr := User{}

	//if c.Request.Method == http.MethodPost {
	//username := c.PostForm("username")
	username := "hello"
	user, err := DB.Query("SELECT * FROM user WHERE username = ? LIMIT 1", username)

	defer user.Close()

	for user.Next() {
		user.Scan(&usr.User_id, &usr.Username, &usr.Email, &usr.Pw_hash)
		break
	}

	if err != nil {
		log.Fatal(err)
	}
	if user == nil {
		fmt.Println("Invalid username")
	} else if !CheckPasswordHash("secret", usr.Pw_hash) {
		fmt.Println("Invalid password")
	} else {
		session.Set("flash", "You were logged in")
		session.Set("user_id", usr.User_id)
		session.Save()
		fmt.Println(":)")
		c.Redirect(http.StatusFound, "/public")
		return
	}
	//}

	c.JSON(http.StatusOK, gin.H{"error": error, "user": usr})
	//c.HTML(http.StatusOK, "login.html", gin.H{"error": error})
}

func register_loadPage(c *gin.Context) {
	var tpl = gonja.Must(gonja.FromFile("template/register.html"))

	req := struct {
		Endpoint string
		Username string
		Email    string
	}{
		Endpoint: c.Request.URL.Path,
	}

	out, err := tpl.Execute(gonja.Context{
		"request": req,
		"error":   "",
	})

	if err != nil {
		fmt.Println(err)
	}

	c.Writer.WriteString(out)
}

func register(c *gin.Context) {

	//fmt.Println(c.PostForm("username"))

	username := c.PostForm("username")
	email := c.PostForm("email")
	password := c.PostForm("password")
	password1 := c.PostForm("password2")
	hash, _ := HashPassword(password)
	errorMessage := ""

	//Check username, email or password is set
	if username == "" {
		errorMessage = errorMessage + " - You have to enter a username \n"
	}
	if !strings.Contains(email, "@") {
		errorMessage = errorMessage + " - You have to enter a valid email address \n"
	}
	if password == "" {
		errorMessage = errorMessage + " - You have to enter a password \n"
	} else {
		//Check the two passwords are the same!
		if password != password1 {
			errorMessage = errorMessage + " - The two passwords do not match \n"
		}
	}

	//Check if user is already there
	existingUser, err := DB.Query("SELECT * FROM user WHERE email = ? ", email)
	defer existingUser.Close()
	if existingUser.Next() {
		errorMessage = errorMessage + " - User was already taken! \n"
	}

	//If no errors then insert in database
	if errorMessage == "" {
		stmt, err := DB.Prepare(fmt.Sprintf("INSERT INTO user (username, email, pw_hash) VALUES('%s', '%s', '%s')", username, email, hash))
		if err != nil {
			fmt.Println(err)
			errorMessage = "err in register"
		}
		defer stmt.Close()

		// Execute the insert statement with the desired values
		result, err := stmt.Exec()
		if err != nil {
			fmt.Println(err)
			errorMessage = "err in register"
		}

		// Get the ID of the inserted record
		id, err := result.LastInsertId()
		if err != nil {
			fmt.Println(err)
			errorMessage = "err in register"
		}
		fmt.Println("Record inserted with ID:", id)
	}

	var tpl *exec.Template
	if errorMessage == "" {
		//c.Redirect(http.StatusAccepted, "/login") HER SKAL VI SLETTE DEN UNDER OG BARE REDIRECTE NÅR LOGIN ER LAVET
		tpl = gonja.Must(gonja.FromFile("template/login.html"))
	} else {
		tpl = gonja.Must(gonja.FromFile("template/register.html"))
	}

	req := struct {
		Endpoint string
		Username string
		Email    string
	}{
		Endpoint: c.Request.URL.Path,
		Username: username,
		Email:    email,
	}

	//Render HTML
	out, err := tpl.Execute(gonja.Context{
		"request": req,
		"error":   errorMessage,
	})

	if err != nil {
		fmt.Println(err)
		return
	}
	c.Writer.WriteString(out)

}

func geturl(email string) string {
	email_md5 := fmt.Sprintf("%s", md5.Sum([]byte(email)))
	hex_md5_email := hex.EncodeToString([]byte(email_md5))

	//format string
	url := fmt.Sprintf("https://www.gravatar.com/avatar/%s?d=identicon&s=48", hex_md5_email)
	return url
}

func main() {
	r := gin.Default()
	store := cookie.NewStore([]byte("secret"))
	r.Use(sessions.Sessions("mysession", store))

	r.LoadHTMLFiles("./template/timeline_go.html")
	r.Static("/static", "./static/")

	ConnectDatabase()

	r.GET("/public", public_timeline)
	r.GET("/login", login)
	r.POST("/login", login)
	r.GET("/register", register_loadPage)
	r.POST("/register", register)

	r.Run()
}
