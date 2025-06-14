# Rate Limiter

A simple rate limiter implementation in Go using the token bucket algorithm. This tool allows you to simulate and test rate limiting with configurable parameters.

## Features

- Token bucket algorithm implementation
- Configurable rate and burst size
- Concurrent request simulation
- Command-line argument support
- Thread-safe implementation

## Installation

```bash
go get github.com/rRateLimit/arg
```

Or clone the repository:

```bash
git clone https://github.com/rRateLimit/arg.git
cd arg
```

## Usage

### Basic Usage

```bash
go run main.go
```

This will run with default parameters:
- Rate: 10 requests/second
- Burst: 20 tokens
- Requests: 50 total requests
- Workers: 5 concurrent workers

### Custom Parameters

```bash
go run main.go -rate 5 -burst 10 -requests 20 -workers 3
```

### Command-line Arguments

- `-rate`: Rate limit in requests per second (default: 10)
- `-burst`: Maximum burst size (token bucket capacity) (default: 20)
- `-requests`: Total number of requests to simulate (default: 50)
- `-workers`: Number of concurrent workers (default: 5)

### Example Output

```
Rate Limiter Configuration:
- Rate: 5 requests/second
- Burst: 10 tokens
- Simulating 20 requests with 3 workers

Worker 0: Processing request 1 at 15:04:05.100
Worker 1: Processing request 2 at 15:04:05.100
Worker 2: Processing request 3 at 15:04:05.100
...

Completed 20 requests in 2.010s
Actual rate: 9.95 requests/second
```

## How It Works

The rate limiter uses a token bucket algorithm:

1. **Token Bucket**: A bucket starts with a configurable number of tokens (burst size)
2. **Token Generation**: Tokens are added to the bucket at the configured rate
3. **Request Processing**: Each request consumes one token
4. **Rate Limiting**: Requests wait when no tokens are available

## Building

To build the binary:

```bash
go build -o ratelimiter main.go
```

Then run:

```bash
./ratelimiter -rate 10 -burst 20
```

## Testing

Run with different configurations to test rate limiting behavior:

```bash
# High rate, small burst - steady flow
go run main.go -rate 100 -burst 10 -requests 100

# Low rate, large burst - initial burst then slow
go run main.go -rate 2 -burst 50 -requests 100

# Equal rate and burst - balanced flow
go run main.go -rate 10 -burst 10 -requests 50
```

## License

MIT License

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.