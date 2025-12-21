package main

import (
	"bufio"
	"fmt"
	"knife/db"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func initializes() {
	dbconn, err := db.InitDB("knife.db")
	if err != nil {
		log.Fatalf("could not connect to database: %v", err)
	}
	defer dbconn.Close()

	fmt.Println("--- Setting up ---")
	fmt.Println("Enter your username:")
	var username string
	fmt.Scanln(&username)

	var display_name string
	fmt.Println("Enter your display name:")
	fmt.Scanln(&display_name)

	var avatar_url string
	fmt.Println("Enter your avatar URL:")
	fmt.Scanln(&avatar_url)

	var bio string
	fmt.Println("Enter your bio:")
	reader := bufio.NewReader(os.Stdin)
	bio, _ = reader.ReadString('\n')
	bio = strings.TrimSpace(bio)

	// Prompt for a password
	var password string
	for {
		fmt.Print("Enter a password for the application: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatalf("Error reading input: %v", err)
		}
		password = strings.TrimSpace(input)

		if len(password) < 8 {
			fmt.Println("Password must be at least 8 characters long. Please try again.")
			continue
		}
		break
	}

	// Hash the password
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Could not hash password: %v", err)
	}

	profileModel := db.NewProfileModel(dbconn)

	var profile db.Profile
	profile.Finger = username
	profile.DisplayName = display_name
	profile.AvatarURL = avatar_url
	profile.Bio = bio
	profile.PasswordHash = string(hash)

	if err := profileModel.Create(&profile); err != nil {
		log.Fatalf("could not create profile: %v", err)
	}

	fmt.Println("Profile created successfully!")
}
