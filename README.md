# Private Cloud Journal (Go + Fyne)

A cross-platform desktop application for creating, encrypting, and syncing secure journals to your personal Google Drive. Built with **Go** (Golang) and **Fyne** GUI toolkit.

## 🔒 Security Architecture

This application prioritizes privacy and security above all else.

* **End-to-End Encryption:** All data is encrypted locally **before** it ever leaves your machine. Google (or anyone with access to your Drive) sees only encrypted binary garbage.
* **Zero-Knowledge Architecture:** The application does not store your password. Your password is hashed (Argon2id) to verify login, but the raw password used for encryption exists only in RAM while the app is running.
* **Metadata Protection:** Notes are stored as encrypted JSON files. Even the file titles and created/modified dates are hidden inside the encrypted payload. The filenames on Google Drive are random IDs (e.g., `a1b2c3d4.json`).
* **Strong Cryptography:**
    * **Encryption:** AES-256-GCM (Authenticated Encryption).
    * **Key Derivation:** Argon2id (Memory-hard password hashing).
    * **Authentication:** Bcrypt (For local password verification).

## 🚀 Features

* **Cross-Platform GUI:** Native look and feel on macOS, Windows, and Linux (via Fyne).
* **Google Drive Sync:** Automatically creates a dedicated `MyNotes` folder in your Drive.
* **Secure Storage:** All notes are saved as encrypted JSON blobs.
* **Offline/Online Hybrid:** Works by caching decrypted notes in memory for fast access.
* **Dark Mode Support:** Fully supports system themes.

## 🛠️ Technology Stack

* **Language:** Go (Golang)
* **GUI Framework:** [Fyne v2](https://fyne.io/)
* **Cloud Storage:** Google Drive API v3
* **Crypto Libraries:** `golang.org/x/crypto` (Argon2, Scrypt, Bcrypt)

## ⚙️ Setup & Installation

### Prerequisites
1.  **Go:** Install Go (1.16 or higher) from [go.dev](https://go.dev/dl/).
2.  **Google Cloud Project:** You need your own credentials to talk to Google Drive.

### Step 1: Google Cloud Setup (One-time)
1.  Go to the [Google Cloud Console](https://console.cloud.google.com/).
2.  Create a **New Project** (e.g., "PrivateCloudJournal").
3.  Search for and **Enable** the **Google Drive API**.
4.  Go to **APIs & Services > OAuth consent screen**:
    * Select **External**.
    * Add your email as a **Test User** (Important!).
5.  Go to **Credentials**:
    * Create **OAuth Client ID** -> **Desktop App**.
    * Download the JSON file.
    * **Rename it** to `credentials.json`.
    * Place `credentials.json` in the root folder of this project.

### Step 2: Install Dependencies
Open your terminal in the project folder and run:
```bash
go mod tidy

```

This will download all necessary Go packages (Fyne, Google API, Crypto, etc.).

### Step 3: Run the App
```bash
go run .
```

# 📖 Usage Guide
## First Run (Registration):

- The app will ask you to create a Master Password.

`
Warning: If you lose this password, your notes are lost forever. There is no reset mechanism.
`

- Upon creation, a browser window will open asking you to grant the app access to your Google Drive.

## Authentication:

- Copy the authorization code from the browser.

- Paste it into your Terminal window where the app is running.

### Managing Notes:

- New Note: Click "New" to clear the editor.

- Save: Click "Encrypt & Save JSON" to encrypt and upload.

- Refresh: Click "Refresh" to download the latest list from Drive.

# 📄 License
This project is licensed under the MIT License - see the LICENSE file for details.

`
Disclaimer: This software is provided "as is", without warranty of any kind. While it uses industry-standard cryptography, the author is not responsible for data loss or security breaches.
`