# Go Distributed & Scalable Link Shortener API
### 📌 Demo & Code

[![Live Demo](https://img.shields.io/badge/Live-Demo-28a745?style=for-the-badge&logo=vercel&logoColor=white)](https://web.cutlink.in/)
[![View on GitHub](https://img.shields.io/badge/View%20on-GitHub-181717?logo=github&style=for-the-badge)](https://github.com/shushantrishav/Go_URL_Shortner)


## 1. Problem Statement

In today’s digital world, long and complicated URLs can be difficult to use, share, and manage. To solve this, there’s a need for a fast, secure, and scalable service that can turn these long URLs into short, memorable, and trackable links. At the same time, the service must protect against misuse and ensure that important data remains available as needed. This project was built to meet those needs by providing a clean, reliable API for creating and managing short URLs — along with useful features like expiration, validation, and rate limiting.

## 2. Technology Stack & Rationale

* **Go (Golang):** This programming language was selected due to its inherent strengths in concurrency management, rapid compilation, static typing, and superior performance. These attributes render Go exceptionally well-suited for the development of high-concurrency network services, such as an API. Furthermore, its integrated HTTP server is characterized by robustness and efficiency.

* **Redis:** Employed as the primary data store, Redis's in-memory, key-value architecture facilitates exceptionally rapid read and write operations. Its native Time-To-Live (TTL) feature directly supports the automatic expiration of links, and its atomic operations are optimally suited for the implementation of a resilient rate limiter.

* **Docker:** Utilized for containerization, Docker provides consistent development, testing, and deployment environments. It effectively encapsulates the application and its dependencies, thereby simplifying the setup process and ensuring cross-platform portability.

* **Gorilla Mux:** As a robust and versatile URL router for the Go ecosystem, Gorilla Mux was chosen for its capacity to manage intricate routing, method matching, and URL path variables, all of which are essential for defining clearly structured API endpoints.

* **Upstash Redis:** This cloud-based, serverless Redis service offers a highly available, scalable, and fully managed Redis instance. Its seamless integration significantly mitigates the operational overhead associated with self-hosting Redis in a production environment.

## 3. High-Level Design (HLD)

The Link Shortener API is architected as a stateless microservice, designed for horizontal scalability.

```
+------------------+     HTTPS     +---------------------+     HTTP/S     +-------------------+
|   Client Device  | <------------>|   Render (Proxy)    | <------------->|   Go API Service  |
| (Browser/cURL)   |               | (HTTPS Termination) |                | (Docker Container)|
+------------------+               +---------------------+                +--------+----------+
                                                                                   |
                                                                                   | TLS
                                                                                   |
                                                                           +-------+--------+
                                                                           | Upstash Redis  |
                                                                           | (Data Storage, |
                                                                           | Rate Limiting) |
                                                                           +----------------+
```

**Workflow:**

1.  **Request Ingress:** Client applications initiate HTTP/HTTPS requests directed towards the public API endpoint.

2.  **Render Proxy (HTTPS Termination):** In deployment scenarios leveraging Render, the platform's infrastructure intercepts the incoming HTTPS request. It subsequently manages the TLS termination process and forwards a standard HTTP request to the internal Go API service.

3.  **Go API Service:**

    * **Rate Limiting:** Upon receipt of a URL shortening request, the service initially consults the Redis-based rate limiter to ascertain whether the client (identified by its IP address) remains within the predefined request limits. Should these limits be exceeded, an `HTTP 429 Method Not Allowed` response is issued.

    * **URL Validation:** Inbound long URLs undergo stringent validation to confirm their adherence to the `https` scheme and to detect the presence of any malicious script injection attempts.

    * **Duplicate Verification:** Prior to the generation of a new short URL, the service queries Redis to determine if the specified `long_url` has been previously shortened. If an existing entry is found, the corresponding `short_url` is returned, and its Time-To-Live (TTL) is extended.

    * **short Generation/Assignment:** In instances where a custom short is not provided, a unique 6-character alphanumeric short is algorithmically generated. Conversely, if a custom short is furnished, its uniqueness is rigorously verified.

    * **Redis Persistence:** The mapping between `(short_url -> long_url)` and `(long_url -> short_url)` is persistently stored within Redis, each with a 7-day TTL.

    * **Response Generation:** A JSON response, encapsulating the `short_url`, the original `long_url`, and the `limit_remaining` for rate limiting purposes, is then transmitted to the client.

4.  **Redirection:** For a `GET` request targeting a short URL, the service retrieves the corresponding `long_url` from Redis and issues an HTTP 301 (Moved Permanently) redirect directive to the client.

5.  **Redis Connectivity:** The Go application establishes and maintains a secure TLS-encrypted connection to the Upstash Redis instance for all data storage and retrieval operations.

## 4. Features

* **Link Shortening:** Facilitates the transformation of extended URLs into concise, easily distributable links.

* **Optional Custom shorts:** Provides users with the capability to specify a custom short code (short) for their link. In the absence of a user-defined short, a unique 6-character alphanumeric string is automatically generated.

* **Automatic Expiration (TTL):** Shortened links are configured to automatically expire and subsequently be removed from the Redis database after a predefined period of 7 days.

* **Repeated Link Handling:** In scenarios where a user attempts to shorten a URL that has previously undergone the shortening process, the existing short URL is returned, and its expiration timestamp is concurrently renewed for an additional 7-day duration.

* **Strict HTTPS Validation:** Incoming URLs are subjected to rigorous validation processes to ensure their adherence to the `https` scheme and to mitigate the risk of malicious JavaScript or PHP injection attempts.

* **Rate Limiting:** Implemented to curtail potential abuse, this feature restricts URL generation to a maximum of 15 URLs per client (identified by IP address) within a 2-minute interval.

* **Enhanced API Response:** The `/shorten` endpoint's response payload has been augmented to include the generated `short_url`, the original `long_url` submitted in the request, and an indicator of the `limit_remaining` within the rate limiting schema.

* **Dockerized Deployment:** The application is encapsulated within Docker containers, thereby enabling streamlined setup, development, and deployment workflows.

* **Environment Variable Configuration:** All pertinent configuration parameters, encompassing Redis address, password, port assignments, and TLS path specifications, are managed through environment variables to enhance security and flexibility.

* **TLS-Secured Redis Connection:** The application is meticulously configured to establish secure, encrypted connections via TLS to external Redis services, such as Upstash.

## 5. API Usage

Upon successful deployment and initiation, the application's functionality is accessible via its defined HTTP/HTTPS endpoints.

### Base URLs

* **Local HTTP:** `http://localhost:8080`

* **Local HTTPS:** `https://localhost:8443` (It is important to note that bypassing browser security warnings for self-signed certificates or utilizing `curl -k` may be necessary).

* **Render Deployment:** `https://your-render-service-name.onrender.com` (Render automatically manages HTTPS termination for deployed services).

### 5.1. Shorten a URL

This endpoint facilitates the conversion of an extended URL into a more concise form, with the option to specify a custom short.

* **Endpoint:** `/shorten`

* **Method:** `POST`

* **Headers:**
    `Content-Type: application/json`

* **Request Body (JSON):**

    **Auto-generated short Example:**

    ```
    {
        "long_url": "https://www.example.com/some/really/long/path/to/a/resource?param1=value1&param2=value2"
    }
    ```

    **Custom short Example:**

    ```
    {
        "long_url": "https://docs.go.dev/doc/effective_go.html](https://docs.go.dev/doc/effective_go.html",
        "custom_short": "effective-go"
    }
    ```

* **Example `curl` Request (Local HTTPS, auto-generated short):**

    ```
    curl -k -X POST \
         -H "Content-Type: application/json" \
         -d '{"long_url": "https://github.com/golang/go"}' \
         https://localhost:8443/shorten
    ```

* **Example Success Response (JSON):**

    ```
    {
        "short_url": "/s/AbC12X",
        "long_url": "https://github.com/golang/go",
        "limit_remaining": 14,
        "message": "URL shortened successfully"
    }
    ```

    Should the `long_url` have been previously shortened, the system will return the pre-existing `short_url` while concurrently extending its Time-To-Live (TTL).

* **Error Responses:**

    * `400 Bad Request`: Issued when the `long_url` is absent, determined to be invalid (e.g., non-HTTPS), or when a `custom_short` provided is already in active use.

    * `429 Too Many Requests`: Returned when the configured rate limit (15 URLs within a 2-minute period) has been surpassed. The response will additionally include the `limit_remaining`.

    * `500 Internal Server Error`: Indicates an unhandled server-side error.

### 5.2. Redirect from Short URL

This endpoint performs a redirection operation from a shortened URL to its original, extended counterpart.

* **Endpoint:** `/s/{short_short}`

* **Method:** `GET`

* **Example `curl` Request (Local HTTPS):**

    ```
    curl -k -L https://localhost:8443/s/AbC12X
    ```

    (The `-L` flag instructs `curl` to automatically follow HTTP redirects).

* **Expected Behavior:** The client's request will be redirected to the original long URL via an HTTP 301 (Moved Permanently) status code.

* **Error Responses:**

    * `404 Not Found`: Occurs if the specified `short_short` does not correspond to an existing entry in the database.

    * `500 Internal Server Error`: Indicates an unhandled server-side error.

### 5.3. Health Check

A diagnostic endpoint provided to ascertain the operational status of the service.

* **Endpoint:** `/health`

* **Method:** `GET`

* **Example `curl` Request:**

    ```
    curl http://localhost:8080/health
    ```

* **Expected Response:** `OK` (accompanied by an HTTP status `200 OK`).

## 6. Local Setup

The following procedures outline the steps required to establish and operate the application within a local development environment utilizing Docker Compose.

### Prerequisites

* **Go (Version 1.22+):** Must be installed locally to execute `go mod` commands.

* **Docker:** Installation and active daemon operation are required.

* **Docker Compose:** Must be installed.

* **OpenSSL:** Installation is necessary for the generation of self-signed certificates.

### Steps

1.  **Repository Cloning:**

    ```
    git clone <your-repository-url>
    cd <your-project-directory>
    ```

2.  **`.env` File Creation:**
    A file named `.env` must be created in the root directory of your project and subsequently populated with the necessary configuration parameters.
    **Note:** It is imperative to replace `your_redis_password` with the actual password associated with your Redis instance. For Upstash, this detail is critically important.

    ```
    REDIS_ADDR=your_redis_addr:6379 # localhost:6379
    REDIS_PASSWORD=your_redis_password
    REDIS_DB=0
    HTTP_PORT=8080
    HTTPS_PORT=8443 # Applicable solely for local HTTPS testing
    TLS_CERT_PATH=/app/localhost.crt # Applicable solely for local HTTPS testing
    TLS_KEY_PATH=/app/localhost.key  # Applicable solely for local HTTPS testing
    ```

3.  **Self-Signed SSL Certificate Generation (for Local HTTPS Testing):**
    These certificates are indispensable for accessing the application via `https://localhost:8443`. Execute the following command within your project's root directory:

    ```
    openssl req -x509 -newkey rsa:4096 -nodes -keyout localhost.key -out localhost.crt -days 365 -subj "/CN=localhost"
    ```

    This operation will yield `localhost.crt` and `localhost.key` files, which will be placed in your project's root directory.

4.  **Go Module Initialization:**
    This step is crucial for ensuring that all project dependencies are downloaded and that `go.mod` and `go.sum` files are correctly configured.

    ```
    go mod init link-shortener # If not previously initialized, substitute 'link-shortener' with your module's designated name.
    go mod tidy
    ```

5.  **Docker Compose Build and Execution:**
    This command initiates the construction of your Go application's Docker image and subsequently launches the containers.

    ```
    docker-compose up --build
    ```

    Should an `Could not connect to Redis: EOF` error be encountered, it is advisable to verify the accuracy of `REDIS_PASSWORD` in your `.env` file and confirm that `TLSConfig` is appropriately passed to both Redis client initializations within `main.go` and `redis/redis.go`.

## 7. Deployment on Render

This application is engineered for straightforward deployment on Render, or comparable platforms that offer integrated Docker support and automatic HTTPS management.

1.  **Code Preparation:**

    * It is essential to ensure that `main.go` is configured to listen on the `PORT` environment variable (which Render dynamically supplies). The local HTTPS setup, involving `localhost.crt` and `localhost.key`, is **not required** for Render deployments.

    * Confirm that your `Dockerfile` **does not** include instructions to copy `localhost.crt` and `localhost.key` into the image.

    * All modifications to your codebase must be committed to a Git repository (e.g., GitHub, GitLab, Bitbucket).

2.  **Render Web Service Configuration:**

    * Navigate to <https://render.com/> and proceed to create a new **Web Service**.

    * Establish a connection to your Git repository.

    * **Name:** Assign a unique identifier to your service.

    * **Runtime:** Select `Docker` (Render typically performs automatic detection).

    * **Build Command:** This field should remain empty.

    * **Start Command:** This field should remain empty.

    * **Plan Type:** Choose the desired service plan.

    * **Environment Variables:** It is **critical** to add the following environment variables within Render's dashboard interface:

        * `REDIS_ADDR`: Your **actual Upstash Redis address**

        * `REDIS_PASSWORD`: Your **actual Upstash Redis password**.

        * `REDIS_DB`: `0`

        * `PORT`: `8080` (This denotes the internal port on which your Go application listens).

    * Initiate the service creation by clicking **"Create Web Service"**.

3.  **Automatic HTTPS Implementation:** Render will automatically provision and administer an SSL/TLS certificate for your service, thereby providing a secure `https://` Uniform Resource Locator (e.g., `https://your-service-name.onrender.com`). Explicit HTTPS configuration within your Go application code is thus rendered unnecessary for Render deployments.

## 8. Important Notes

* **Redis Password Security:** It is paramount to maintain the confidentiality of your `REDIS_PASSWORD` and to abstain from hardcoding this credential. Adherence to environment variable-based management, as herein demonstrated, is strongly recommended.

* **Local HTTPS Warnings:** During local testing via `https://localhost:8443`, web browsers will typically issue security warnings due to the self-signed nature of the certificate. It is incumbent upon the user to bypass these warnings. For `curl` utility usage, the `-k` or `--insecure` flag should be employed.

* **Logging Practices:** Application logs are directed to `stdout` and `stderr` streams, which are subsequently captured by Docker and Render's respective logging infrastructures, and are accessible from their administrative dashboards. These logs are not directly exposed to the frontend interface.

* **Rate Limiter Key Considerations:** The current rate limiting mechanism utilizes the client's `r.RemoteAddr` (IP address) as the primary key. For robust production deployments (e.g., behind a load balancer that may modify `X-Forwarded-For` headers), or to implement rate limiting on a per-user basis, it may be necessary to adapt this approach to leverage a unique user identifier derived from an authentication system.
* **Cross-Origin Resource Sharing (CORS) Configuration:**
The API incorporates CORS middleware (github.com/rs/cors) to manage cross-origin requests from web browsers. For development purposes, http://localhost:5500 is explicitly permitted as an allowed origin. In production environments, it is imperative to configure AllowedOrigins to exclusively include the precise URL(s) of your frontend application to maintain optimal security. The current configuration explicitly allows requests from:

   * http://localhost:5500 (for local frontend development)

   * The API is configured to allow GET, POST, and OPTIONS methods, and the Content-Type header, with credentials allowed. This setup ensures that your frontend application can communicate securely with the API across different origins.

## 9. Scope for Future Development

* **User Authentication:** The integration of user registration and login functionalities to enable individual users to manage their respective shortened links.

* **Custom Domain Support:** The provision of capabilities allowing users to employ their own domain names for short links (e.g., `short.mydomain.com/abc`).

* **Link Analytics:** The implementation of tracking mechanisms for click counts, referrer information, geographical location data, and other pertinent metrics associated with shortened links.

* **QR Code Generation:** The automated generation of Quick Response (QR) codes for shortened URLs.

* **Link Deactivation/Deletion:** The provision of an API endpoint facilitating the deactivation or permanent removal of shortened links by authorized users.

* **Admin Dashboard:** The development of a simplified web-based interface for administrative oversight of links and monitoring of service health.

* **Advanced Rate Limiting:** The exploration and implementation of more sophisticated rate limiting strategies, such as per-user limits or burst limits.

* **Database Abstraction:** The introduction of a database abstraction layer to support alternative database systems (e.g., PostgreSQL, MongoDB) for the long-term persistence of link metadata.

* **Health Checks and Monitoring:** Integration with tools such as Prometheus and Grafana for comprehensive application metrics and real-time alerting capabilities.

* **Graceful Shutdown:** The implementation of robust graceful shutdown procedures for the Go application to ensure orderly termination and data integrity.

---

### 📌 Demo & Code

[![Live Demo](https://img.shields.io/badge/Live-Demo-28a745?style=for-the-badge&logo=vercel&logoColor=white)](https://web.cutlink.in/)
[![View on GitHub](https://img.shields.io/badge/View%20on-GitHub-181717?logo=github&style=for-the-badge)](https://github.com/shushantrishav/Go_URL_Shortner)

```