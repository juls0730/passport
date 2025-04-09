# Passport

Passport is a simple, fast, and lightweight web dashboard/new tab replacement.

> "I cant believe I've never broken it" - me

## Getting Started

![Screenshot of Passport](/screenshot.png)

### Prerequisites

- [ZQDGR](https://github.com/juls0730/zqdgr)
- [Go](https://go.dev/doc/install)
- [sqlite3](https://www.sqlite.org/download.html)
- [TailwdinCSS CLI](https://github.com/tailwindlabs/tailwindcss/releases/latest)

## Usage

1. Clone the repository
2. Configure the `.env` file, an example is provided in the `.env.example` file, see below for every available environment variable
4. Deploy `passport` to your web server
5. profit

### Configuration

#### Passport configuration

| Environment Variable | Description | Required | Default | 
| --- | --- | --- | --- | 
| `PASSPORT_DEV_MODE` | Enables dev mode | false | false | 
| `PASSPORT_ENABLE_PREFORK` | Enables preforking | false | false |
| `PASSPORT_ENABLE_WEATHER` | Enables weather data, see [Weather configuration](#weather-configuration) | false | false |
| `PASSPORT_ENABLE_UPTIME` | Enables uptime data, see [Uptime configuration](#uptime-configuration) | false | false |
| `PASSPORT_ADMIN_USERNAME` | The username for the admin dashboard | true |
| `PASSPORT_ADMIN_PASSWORD` | The password for the admin dashboard | true |
| `PASSPORT_SEARCH_PROVIDER` | The search provider to use for the search bar, without any query parameters | true |
| `PASSPORT_SEARCH_PROVIDER_QUERY_PARAM` | The query parameter to use for the search provider, e.g. `q` for most providers |false | q |

#### Weather configuration

| Environment Variable | Description | Required | Default | 
| --- | --- | --- | --- | 
| `OPENWEATHER_PROVIDER` | The weather provider to use, currently only `openweathermap` is supported | true | openweathermap | 
| `OPENWEATHER_API_KEY` | The OpenWeather API key | if enabled | | 
| `OPENWEATHER_TEMP_UNITS` | The temperature units to use, either `metric` or `imperial` | false | metric |
| `OPENWEATHER_LAT` | The latitude of your location | if enabled | | 
| `OPENWEATHER_LON` | The longitude of your location | if enabled | | 
| `OPENWEATHER_UPDATE_INTERVAL` | The interval in minutes to update the weather data | false | 15 | 

#### Uptime configuration

| Environment Variable | Description | Required | Default | 
| --- | --- | --- | --- | 
| `UPTIMEROBOT_API_KEY` | The UptimeRobot API key | if enabled | | 
| `UPTIMEROBOT_UPDATE_INTERVAL` | The interval in seconds to update the uptime data | false | 300 | 

### Adding links and categories

The admin dashboard can be accessed at `/admin`, you will be redirected to the login page if you are not logged in, use the credentials you configured in the `.env` file to login. Once logged in you can add links and categories.

## License

This project is licensed under the BSL-1.0 License - see the [LICENSE](LICENSE) file for details
