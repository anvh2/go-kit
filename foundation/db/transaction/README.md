# Distributed Transactions & Patterns in Microservices

## 1. Introduction
Distributed transactions occur when multiple services or databases need to maintain **consistency** across operations. Traditional single-database ACID transactions are insufficient in distributed systems, so alternative patterns are used to ensure reliability, consistency, and fault tolerance.

Key challenges include:
- Network failures
- Node crashes
- Latency
- Maintaining atomicity across services

---

## 2. Two-Phase Commit (2PC)

### 2.1 Overview
Two-Phase Commit is a protocol to achieve **atomic commits across distributed systems**. It consists of two phases:

1. **Prepare Phase**
   - Coordinator asks all participants to prepare to commit.
   - Each participant writes the transaction log and responds with **OK** or **FAIL**.

2. **Commit Phase**
   - If all participants are OK, coordinator sends **commit** command.
   - If any participant fails, coordinator sends **rollback**.

### 2.2 Advantages
- Strong consistency (all-or-nothing commit)
- Simplicity in concept

### 2.3 Disadvantages
- **High latency** due to multiple communication rounds
- **Blocking**: participants wait for coordinator in case of failure
- **Single point of failure**: if coordinator crashes, participants may be left in uncertain state
- Poor scalability for high-throughput microservices

### 2.4 Usage
- Rarely used in modern microservices
- Mostly in traditional databases or tightly coupled systems

---

## 3. SAGA Pattern

### 3.1 Overview
SAGA is an **event-driven pattern** to handle distributed transactions **without locking resources across services**.

- Each service executes a **local transaction**
- If a step fails, **compensating transactions** are triggered to undo previous steps
- Supports **eventual consistency**

### 3.2 Implementation Approaches
1. **Choreography**
   - Services emit events that trigger the next step in the saga
   - No central coordinator
2. **Orchestration**
   - A central orchestrator directs each step of the saga
   - Handles failures and triggers compensations

### 3.3 Advantages
- Non-blocking, resilient to partial failures
- Scalable for high-throughput microservices

## 4. Outbox Pattern

### 4.1 Overview
The Outbox pattern ensures **reliable message delivery** as part of a local transaction:

- During a local DB transaction, write the event/message into an **outbox table**
- A background process reads the outbox table and publishes messages to message queues (Kafka, Pub/Sub)
- Ensures **at-least-once delivery** and prevents message loss

### 4.2 Advantages
- Combines local transaction with messaging reliably
- Works well with SAGA to coordinate multi-service workflows
- Reduces complexity compared to 2PC
