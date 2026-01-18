# Authorization

Prabogo supports multiple authorization strategies to secure your applications. You can choose between internal bearer key authentication or integrate with external identity providers like Authentik for enhanced security.

## Configuration Overview

Authorization is configured through the `AUTH_DRIVER` environment variable. Prabogo supports two main authorization methods:

## 1. Internal Bearer Key Authentication

This method uses internal client bearer keys stored in your database. While simple to implement, it's recommended to use mTLS (mutual TLS) when using this approach for enhanced security.

### Configuration

```bash
# Set AUTH_DRIVER to empty/blank for internal authentication
AUTH_DRIVER=

# Other required configurations
DATABASE_USERNAME=prabogo
DATABASE_PASSWORD=prabogo
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_NAME=prabogo
```

### Security Recommendations

> **⚠️ Security Note:** When using internal bearer key authentication, it's highly recommended to implement mTLS (mutual TLS) for additional security. This ensures both client and server authentication through certificates.

Benefits of using mTLS with internal authentication:

- Mutual authentication between client and server
- Certificate-based identity verification
- Protection against man-in-the-middle attacks
- Enhanced encryption for data in transit

## 2. Authentik JWT Authentication (Recommended)

For enhanced security, Prabogo integrates with Authentik using JWT (JSON Web Token) client credentials flow. This method provides enterprise-grade authentication and authorization capabilities.

### Configuration

```bash
# Set AUTH_DRIVER to "authentik" for JWT authentication
AUTH_DRIVER=authentik

# Configure the JWKS URL for token validation
AUTH_JWKS_URL=https://your-authentik-server.com/application/o/your-app/jwks/

# Example configuration
# AUTH_JWKS_URL=https://authentik-server.prabogo.orb.local/application/o/prabogo/jwks/
```

### Authentik Setup Requirements

To use Authentik JWT authentication, you need to:

1. **Run Authentik Server (Development)**

   For development and testing purposes, you can run Authentik locally using the provided Docker Compose file:

   ```bash
   # Start Authentik services (PostgreSQL, Redis, Server, Worker)
   docker-compose -f docker-compose.authentik.yml up -d

   # Check if all services are running
   docker-compose -f docker-compose.authentik.yml ps

   # Access Authentik web interface
   # Navigate to: http://localhost:9080
   ```

   **Default Access:**
   - **HTTP:** `http://localhost:9080`
   - **Initial Setup:** `http://localhost:9080/if/flow/initial-setup`

   > **💡 Development Note:** The docker-compose.authentik.yml file includes all necessary services (PostgreSQL on port 5433, Redis on port 6380, and Authentik server). Make sure these ports are available on your system.

2. **Create an Application in Authentik**
   - Configure OAuth2/OpenID provider
   - Configure Application
   - Configure appropriate scopes and permissions (necessary)

3. **Create Service Account (Recommended)**
   - Create a user with **service account** type in Authentik
   - Generate an **app password** for the service account
   - Use username/password authentication to obtain JWT tokens
   - This approach provides better security and token management compared to client credentials

   Once you have created a service account and app password, you can obtain JWT tokens using a curl request:

   ```bash
   curl --location 'https://your-authentik-server.com/application/o/token/' \
   --header 'Content-Type: application/x-www-form-urlencoded' \
   --data-urlencode 'grant_type=client_credentials' \
   --data-urlencode 'client_id=your-client-id' \
   --data-urlencode 'scope=openid' \
   --data-urlencode 'username=your-service-account' \
   --data-urlencode 'password=your-generated-app-password'
   ```

   **Response:** The API will return a JSON response containing the access_token (JWT) and other token information:

   ```json
   {
     "access_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6IjE2NzM...",
     "token_type": "Bearer",
     "expires_in": 300,
     "id_token": "eyJhbGciOiJSUzI1NiIsImtpZCI6IjE2NzM..."
   }
   ```

4. **Configure JWKS Endpoint**
   - Obtain the JWKS URL from your Authentik application
   - Ensure the endpoint is accessible from your Prabogo application

5. **Set Environment Variables**
   - Update `AUTH_DRIVER=authentik`
   - Set the correct `AUTH_JWKS_URL`

### Benefits of Authentik JWT Authentication

- **Enhanced Security**: Industry-standard JWT tokens with digital signatures
- **Centralized Identity Management**: Single sign-on (SSO) capabilities
- **Scalability**: Stateless authentication suitable for microservices
- **Token Validation**: Automatic token validation using JWKS
- **Fine-grained Access Control**: Role-based and attribute-based access control
- **Audit Trail**: Comprehensive logging of authentication events

## Choosing the Right Method

| Feature | Internal Bearer Key | Authentik JWT |
|---------|-------------------|---------------|
| Setup Complexity | Simple | Moderate |
| Security Level | Good (with mTLS) | Excellent |
| Scalability | Limited | High |
| External Dependencies | Database only | Authentik server |
| Token Management | Manual | Automatic |
| SSO Support | No | Yes |

## Migration Between Methods

You can easily switch between authentication methods by updating the environment variables:

```bash
# Switch to internal authentication
AUTH_DRIVER=
# Remove or comment out AUTH_JWKS_URL

# Switch to Authentik authentication
AUTH_DRIVER=authentik
AUTH_JWKS_URL=https://your-authentik-server.com/application/o/your-app/jwks/
```

After changing the configuration, restart your application to apply the new authentication method.