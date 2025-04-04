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
2. Configure the `.env` file, an example is provided in the `.env.example` file, see below for every available environment variable
4. Deploy `passport` to your web server
5. profit

#### Configuration

| Environment Variable | Description | Required | Default |
| --- | --- | --- | --- | 
| `PASSPORT_DEV_MODE` | Enables dev mode | false | false |
| `PASSPORT_ENABLE_WEATHER` | Enables weather data, requires an OpenWeather API key | false | true |
| `PASSPORT_ENABLE_UPTIME` | Enables uptime data, requires an UptimeRobot API key | false | true |
| `PASSPORT_ADMIN_USERNAME` | The username for the admin dashboard | true |
| `PASSPORT_ADMIN_PASSWORD` | The password for the admin dashboard | true |
| `PASSPORT_SEARCH_PROVIDER` | The search provider to use for the search bar | true |
| `OPENWEATHER_API_KEY` | The OpenWeather API key | if enabled |
| `OPENWEATHER_LAT` | The latitude of your location | if enabled |
| `OPENWEATHER_LON` | The longitude of your location | if enabled |
| `OPENWEATHER_UPDATE_INTERVAL` | The interval in minutes to update the weather data | false | 15 |
| `UPTIMEROBOT_API_KEY` | The UptimeRobot API key | if enabled |
| `UPTIMEROBOT_UPDATE_INTERVAL` | The interval in seconds to update the uptime data | false | 300 |

### Adding links and categories

The admin dashboard can be accessed at `/admin`, you will be redirected to the login page if you are not logged in, use the credentials you configured in the `.env` file to login. Once logged in you can add links and categories.

## License

This project is licensed under the BSL-1.0 License - see the [LICENSE](LICENSE) file for details
