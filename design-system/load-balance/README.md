# Load Balancer

- **Scalability**: As traffic grows, you can add more servers behind the load balancer without redesigning your entire architecture.
- **High Availability**: If one server goes offline or crashes, the load balancer automatically reroutes traffic to other healthy servers.
- **Performance Optimization**: Balancing load prevents certain servers from overworking while others remain underutilized.
- **Maintainability**: You can perform maintenance on individual servers without taking your entire application down.

## Routing Algorithm

The load balancer decides which server should get the request. Common routing algorithms include:

- **Round Robin**: Requests are distributed sequentially to each server in a loop.
- **Weighted Round Robin**: Each server is assigned a weight (priority). Servers with higher weights receive proportionally more requests.
- **Least Connections**: The request goes to the server with the fewest active connections.
- **IP Hash**: The load balancer uses a hash of the client’s IP to always route them to the same server (useful for sticky sessions).
- **Random**: Select a server randomly (sometimes used for quick prototypes or specialized cases).

## Key Features and Considerations

### SSL/TLS Termination

Offloads cryptographic operations from the servers. The load balancer decrypts incoming SSL traffic and sends plain HTTP traffic to backend servers, reducing their CPU load.

### Sticky Sessions (Session Persistence)

Ensures a client’s subsequent requests are always routed to the same server. This can be important for stateful applications (though it’s often better to design stateless services and store state externally, e.g., in a cache).

### Auto Scaling

In cloud environments, you can integrate the load balancer with an auto-scaling group. As traffic increases, new servers spin up and automatically register with the load balancer.

### Caching and Compression

Some load balancers (especially application-level) can cache responses or compress them to reduce bandwidth.

### Security Features

Modern load balancers often support Web Application Firewall (WAF) capabilities, DDoS protection, and more.