# Passport

Passport is a simple, fast, and lightweight web dashboard/new tab replacement.

> "I cant believe I've never broken it" - me

## Getting Started

![Screenshot 2025-03-28 at 07-44-22 Passport](https://github.com/user-attachments/assets/d31b0694-3445-46f8-af01-158703e44b4c)

### Prerequisites

- [ZQDGR](https://github.com/juls0730/zqdgr)
- [Go](https://go.dev/doc/install)
- [sqlite3](https://www.sqlite.org/download.html)
- [TailwdinCSS CLI](https://github.com/tailwindlabs/tailwindcss/releases/latest)

### Usage

1. Clone the repository
2. Configure the `.env` file, an example is provided in the `.env example` file
   - The `OPENWEATHER_API_KEY` is required for the weather data to be displayed, if you want to disable the weather data, set `PASSPORT_ENABLE_WEATHER` to `false`
   - The `OPENWEATHER_LAT` and `OPENWEATHER_LON` are required for the weather data to be displayed
   - The `PASSPORT_ADMIN_USERNAME` and `PASSPORT_ADMIN_PASSWORD` are required for the admin dashboard
   - The `PASSPORT_SEARCH_PROVIDER` is the search provider used for the search bar, %s is replaced with the search query
   - The `UPTIMEROBOT_API_KEY` is required for the uptime data to be displayed, if you want to disable the uptime data, set `PASSPORT_ENABLE_UPTIME` to `false`
3. Run `zqdgr build` to build a standalone binary
4. Deploy `passport` to your web server
5. profit

### Adding links and categories

The admin dashboard can be accessed at `/admin`, you will be redirected to the login page if you are not logged in, use the credentials you configured in the `.env` file to login. Once logged in you can add links and categories.

## License

This project is licensed under the BSL-1.0 License - see the [LICENSE](LICENSE) file for details
