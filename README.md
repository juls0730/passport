# Passport

Passport is a simple, fast, and lightweight web dashboard/new tab replacement.

## Getting Started

![Screenshot 2024-11-11 at 00-20-06 Passport](https://github.com/user-attachments/assets/ba16da2f-fb84-4f45-827f-3de0af6626a0)

### Prerequisites

- [Go](https://go.dev/doc/install)
- [sqlite3](https://www.sqlite.org/download.html)
- [TailwdinCSS CLI](https://github.com/tailwindlabs/tailwindcss/releases/latest)

### Usage

1. Clone the repository
2. Configure the `.env` file, an example is provided in the `.env example` file
   - The `OPENWEATHER_API_KEY` is required for the weather data to be displayed
   - The `OPENWEATHER_LAT` and `OPENWEATHER_LON` are required for the weather data to be displayed
   - The `PASSPORT_ADMIN_USERNAME` and `PASSPORT_ADMIN_PASSWORD` are required for the admin dashboard
   - The `PASSPORT_SEARCH_PROVIDER` is the search provider used for the search bar, %s is replaced with the search query
3. Run `sqlite3 passport.db < passport.sql` to create the database
4. Run `go build` to build the project
5. Deploy passport, passport.db and .env, and preferably the public folder (but you dont have to) to your web server
6. profit

### Adding links and categories

The admin dashboard can be accessed at `/admin`, you will be redirected to the login page if you are not logged in, use the credentials you configured in the `.env` file to login. Once logged in you can add links and categories.

## License

This project is licensed under the BSL-1.0 License - see the [LICENSE](LICENSE) file for details
