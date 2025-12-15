package main

import (
	"fmt"
	"knife/db"
	"log"
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
	fmt.Scanln(&bio)

	profileModel := db.NewProfileModel(dbconn)

	var profile db.Profile
	profile.Finger = username
	profile.DisplayName = display_name
	profile.AvatarURL = avatar_url
	profile.Bio = bio

	if err := profileModel.Create(&profile); err != nil {
		log.Fatalf("could not create profile: %v", err)
	}

	fmt.Println("Profile created successfully!")
}
