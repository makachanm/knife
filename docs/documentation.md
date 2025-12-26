# Knife Documentation

## Overview

Knife is a lightweight, single-user blogging platform implementing the **ActivityPub** protocol. 
It allows you to publish notes, send it to fediverse instances (like Mastodon, Misskey, etc.).

It is built with **Go** and uses **SQLite** for data storage, making it easy to deploy and maintain.

## Features

-   **ActivityPub Federation**:
    -   Send posts (Notes) to your followers.
-   **Note Management**:
    -   Support for **Content Warnings (CW)** with foldable UI.
    -   Visibility levels: Public, Unlisted, Followers Only, Private.
    -   Markdown support for content.
-   **Categories**:
    -   Organize notes into categories.
-   **Bookmarks**:
    -   Bookmark notes for later reading.
    -   Manage your bookmarks list.
-   **Profile**:
    -   Simple customizable profile with display name, bio.
    -   View recent posts on your profile.
-   **Drafts**:
    -   Auto-save drafts while writing new notes.
-   **Simple Frontend**:
    -   Clean, responsive HTML/CSS/JS frontend.
    -   No complex framework used in frontend (vanilla JS).

## Getting Started

### Prerequisites

-   Go 1.18+
-   GCC (for SQLite cgo driver)

### Installation

1.  Clone the repository:
    ```bash
    git clone https://github.com/makachanm/knife.git
    cd knife
    ```

2.  Build the application:
    ```bash
    go build
    ```

3.  Initialize the application (first run):
    ```bash
    ./knife setup
    ```
    Follow the prompts to set up your username, password, and profile details.

4.  Generate keys (optional, usually handled by setup/run):
    ```bash
    ./knife initkey
    ```

### Running

Start the server:
```bash
./knife
```

Access the application at `http://localhost:8080`.

## API Endpoints

### Local API

-   `GET /api/notes`: List recent notes.
-   `POST /api/notes`: Create a new note.
-   `GET /api/notes/{id}`: Get a specific note.
-   `DELETE /api/notes/{id}`: Delete a note.
-   `GET /api/category`: List all categories.
-   `GET /api/category/{name}`: List notes in a category.
-   `GET /api/profile`: Get profile info.
-   `PUT /api/profile`: Update profile info.
-   `GET /api/bookmarks`: List bookmarks.
-   `POST /api/bookmarks`: Add a bookmark.
-   `DELETE /api/bookmarks/{id}`: Remove a bookmark.

### ActivityPub Endpoints

-   `/.well-known/webfinger`: WebFinger discovery.
-   `/profile`: Actor profile (Accept: application/activity+json).
-   `/inbox`: Inbox for receiving activities (POST).
-   `/notes/{id}`: Note object (Accept: application/activity+json).

## License

[zlib License](LICENSE)
