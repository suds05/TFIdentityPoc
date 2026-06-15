# Implementing a Global Service with High Availability and Low Latency
We want to evolve an architecture for a Global Directory used in a distributed system
with high availability and low latencies across the globe.

## Background
In most big multi-tenant systems, there are some top-level entities that are used acoss the system.
Example: Users, Tenants, Organizations, Workspaces, Teams, etc.

This pattern is common and can be seen in multiple systems:
1. In Exchange Online, there are Users, Organizations and Mailboxes.
2. In Azure Resource Manager, there are Subscriptions, Resource Groups and Resources.

These top-level entities are typically used for discovery, authorization and routing. They answer questions such as:
- Set of all Users in an Organization.
- Set of Organizations a User is part of.
- What Mailboxes can a user access.
- Which Capacity Unit hosts which Mailbox.
- URL to the Capacity Unit / Mailbox.

Entities underneath the top-level entities are typically homed in one of many Capacity Units
that are deployed in several regions.

**The harder question we try to answer here is, how do we implement a Global Service
to support these top-level entities?**

These are accessed from multiple Capacity Units and Clients from various regions. In which regions do we deploy Global Service stamps? How do we ensure their availability? And how do we reduce geo-latency to access them?

## Requirements
1. The Global Service (and DB) must be hosted so that geo-latency to reach it is minimal
for Clients and Capacity Units. Callers can be present anywhere in the world, and we
would prefer calls to be served from a region close to them.

Some of the top-level objects have a geo affinity. E.g., Users have addresses they typically
work from. Organzation have data residency requirements within a region. Teams are
typically operate from a office in one location, etc.

Design must exploit such affinities to give optimal latency. But work globally otherwise too.

2. The Global Service (and DB) must remain highly available across infrastructure
failures, including:
- Availability zone failure
- Single Region failure
- Single Cloud provider failure
In essense, the deployment must be multi-region and multi-cloud.

3. Global Service should be optimal for a read-heavy workload.
The top level objects are typically read more often than written. Read scenarios include:
- Authorization lookup: determine whether a user has access to a team.
- Routing lookup: determine which Capacity Unit hosts a team.

Writes are less frequent and typically occur during Provisioning and Management
operations, such as:
- inviting a user to a team
- removing a user from a team
- creating or deleting a team
- moving a team between Capacity Units
- updating Capacity Unit metadata

Design should be optimal for former, and reasonable for latter.

4. We absolutely want to avoid data correctness issues while provisioning or management operations.
We can tolerate some latency (design parameter to be minimized), but we do not want lost updates, conflicting updates, or other correctness issues even when access across regions.


## Architectural models
Here, we use the following terminology:
- The Global Service is the logical service/API for top-level entities.
- A Global Service stamp is a deployed copy of the Global Service in a specific
  region / provider / AZ scope, behind regional routing or load balancing.
- A Global Service instance is a single running pod, VM, or process inside a
  Global Service stamp.
- The Global DB is the logical database used by the Global Service.
- A Global DB cluster is a physical database cluster that houses the Global DB.
- A GeoArea is a area of geographic affinity in the application, such as NAM, EUR, or APC.
- A GeoAreaId is the row or document field that identifies the GeoArea for an object.
- A GeoArea can be implemented using multiple physical database shards
- An Owner DB cluster is the Global DB cluster currently allowed to write a GeoArea.

The Global Service compute layer is stateless. Global Service stamps can be
deployed in multiple regions / providers, and are typically deployed in several
edge locations to remain close to users.

The deployment and usage of Global DB clusters is the trickier part of the design.

We will look at two architectural models. We will assess implementing each with suitable member from the broad SQL and NoSQL families:
1. MongoDB / Mongo Atlas for NoSQL
2. CockroachDB / Cockroach Cloud for SQL.


### Model A: Multi-Active GeoArea DB Clusters
Here, we will deploy Global DB clusters in each Geographic Area (`GeoArea`) we want to support. Each Global DB cluster will be multi-AZ single-region hosted by some cloud provider in that geographical area (GeoArea).

For instance, we could support North America (NAM), Europe (EUR), Asia Pacific (APC) as our three GeoAreas. We can correspondingly have three Global DB clusters, say in us-west1, europe-west1 and asia-southeast2. Each Global DB cluster could use 3 Availability Zones within the region. We can use different cloud providers for different Global DB clusters. Cross-provider support is a big advantage in this model.

Global Service stamps can be deployed in many more regions - say us-west1, us-west2, europe-west1, europe-east1, etc. The Global Service instances in those stamps will all use these three Global DB clusters.

All rows or documents in any table or collection of the global data will be tied
to a `GeoAreaId`. Location aware objects like User (or Team) can be tied to a
GeoArea during provisioning based on user's location. Location independent objects
can be placed in a GeoArea based on some consistent hashing.

All GeoAreas will be present in all Global DB clusters, and all Global DB clusters are active. Hence the name: `Multi-Active GeoArea DB Clusters`.

However, Write operations for a GeoArea will happen only from one Owner DB cluster. **This is the key design invariant that preserves data consistency**. While all Global DB clusters are active and all of them can handle traffic, update for a row or document will happen only at the Owner DB cluster. We will have a mapping of GeoAreaId to Owner DB cluster as a metadata table.

Read operations for a GeoArea can happen from any Global DB cluster. However, Global Service instances
will preserve affinity for a client: if an instance used a Global DB cluster for a client recently,
the instance will use the same cluster. That way, when an update was recently done, subsequent requests from client
go to same Global DB cluster (for read-your-own-writes)

```mermaid
graph
  CL[Client]
  CL-->C1
  CL-->S
  C1[Service Stamp NAM1]
  U1[NAM Global DB Cluster]
  U2[EUR Global DB Cluster]
  U3[APC Global DB Cluster]
  S[Service Stamp EUR1]
  C1-->U1
  C1-.->U2
  C1-.->U3
  S-.->U1
  S-->U2
  S-.->U3
```
#### Geo-latency
Client call will land on the Global Service stamp that is closest in terms of geo-latency
via some Global External Load Balancer or Frontdoor.

For read requests, the serving Global Service instance will invoke the Global DB cluster that's closest. This
can again be done by some Load Balancer via Shared URL. E.g.,
any Global Service instance in a NAM stamp will end up calling NAM Global DB cluster for low latencies.

For write requests, the serving Global Service instance will explicitly invoke the Owner DB cluster
that owns the GeoArea in question. This may be a relatively expensive call.

#### Replication
Replication across GeoArea DB clusters will be Asynchronous. Note that this replication has to be:
1. Bidirectional. Any two Global DB clusters will exchange CDC events for GeoAreas they write to.
2. Multi-way. A Global DB cluster could send and receive events from more than one Global DB cluster.
This type of Bidirectional Multi-Way replication is not supported natively by Database systems.

So we will build them over CDC Events.
* This replication will be managed by the Application (A Sync background service).
* We will gather DB level CDC events for each Global DB cluster.
* We use some PubSub infra to disseminate and apply these events
* Sync service from a Global DB cluster publishes events into its Topic/Queue
* Sync service from Global DB clusters subscribe to this Topic, and apply to their DB.
* Since any row can only be updated in one Owner DB cluster (whichever Global DB cluster serves that row's
  GeoArea), conflicts will not happen.
* Every entity will have a monotonically increasing Revision number to run
  consistency checks and do reconciliation.

```mermaid
graph
  U1[NAM Global DB Cluster]
  U2[EUR Global DB Cluster]
  U3[APC Global DB Cluster]
  S[Replication Infra]
  U1-.->S
  U2-.->S
  U3-.->S
  S-.->U1
  S-.->U2
  S-.->U3
```

#### Failure model and Failovers
In this model, Global Service instance failure is transparent. A client call will reach
another healthy instance in the same stamp. Global Service stamp failure is also transparent
because the client call will reach a different healthy stamp via Global Load Balancer or
Frontdoor. The Global Service is truly Active-Active.

Single Availability Zone(AZ) Failure can be tolerated in a Global DB cluster, as it spans three AZs.

If there is a Region Failure, the corresponding Global DB cluster will be down.
  * However Read traffic can continue to be served by some other Global DB cluster.
  * Write traffic to unaffected GeoAreas continues to get served.
  * For Failover, the failed Global DB cluster is put in maintenance mode and Metadata tables
    mapping GeoAreaId to Owner DB cluster need to be updated. Then Global Service instances will
    go to the new Owner DB cluster for affected GeoAreas.

Cloud Provider failure will be similar to Region failure. Some Global DB clusters may go down.
But other Global DB clusters (using different Cloud Providers) will serve Read Traffic.
And Failover can be done to these Global DB clusters for affected GeoAreas.

### Model B: Managed Global Database
In this model, we try to push the replication into the database itself by
creating a managed multi-region Global DB cluster.

Global Service stamps are deployed close to clients in all/multiple regions.
All Global Service instances connect to one managed Global DB cluster that's spread across
at least 3 regions, and 6 AZs.

```mermaid
graph LR
  subgraph NAM
    direction TB
    NAM_AP["A_Primary"]
    NAM_AV1["A_Voting"]
    NAM_AV2["A_Voting"]
    NAM_CR["C_Read"]
    NAM_BR["B_Read"]
  end

  subgraph EUR
    direction TB
    EUR_AR["A_Read"]
    EUR_CR["C_Read"]
    EUR_BP["B_Primary"]
    EUR_B1["B_Voting"]
    EUR_B2["B_Voting"]
  end

  subgraph APC
    direction TB
    APC_CP["C_Primary"]
    APC_CV1["C_Voting"]
    APC_AR["A_Read"]
    APC_BR["B_Read"]
    APC_CV2["C_Voting"]
  end
```

In the above example:
1. GeoArea A has its primary and voting replicas in NAM.
2. GeoArea A also has its read-only non-voting replicas in EUR and APC.
3. Replication between the nodes is handled by the database.
4. Automatic failover within the Voting nodes is handled by the database.

In this model, the replication and consistency is handled by the database. This
is a big benefit.

The primary limitation will be on scaling. The maximum number of Zones or GeoAreas
can be limited by the Database tech (e.g., 9 Zones in Mongo)

#### MongoDB
We will use Global Clusters as described here: https://www.mongodb.com/docs/atlas/global-clusters/. Mongo Global cluster
has concept of a `Zone` that nicely maps to our GeoArea.

Each row or document in the table is associated with a GeoArea as before. In MongoDB, we map these GeoAreas
to Zones. Thus we'll have three Zones:
* NAM
* EUR
* APC

For each Zone (and GeoArea), we will pick one region where voting replicas will live.
Writes happen in this region alone. Say, for NAM, we use us-west1 in AWS.
For EUR, we use europe-west2 in GCP. And for APC, we use westindia in Azure.
**Note: The Voting replicas should be mapped to one region in one cloud provider.
If they are spread across regions, writes can become slow**

We will add Non-voting replicas in each of the other GeoAreas. This will make sure
Reads are low latency globally.

![Zone configuration in Mongo Atlas](zone_config_mongo_atlas.png)

![Zone to region mapping in Mongo Atlas](zone_region_mapping_mongo_atlas.png)

Global Cluster can support up to nine distinct Zones. Zone Scaling is a limitation in this architecture.

#### CockroachDB
CockroachDB also supports global database instead of separate per-GeoArea databases with application-managed replication.

For table placement, there are two useful patterns:
* `REGIONAL BY ROW` for rows that have a clear home GeoArea, such as user-team
  membership and team routing records. This keeps reads and writes for a
  homed user or team close to that GeoArea.
* `GLOBAL` for tables that need low-latency reads from all regions in the cluster.
  Writes to `GLOBAL` tables have higher latency and should be reserved for read-mostly
  data.

In our case, we can model the Users and Teams as GLOBAL table. This will give low latency reads, but writes will be somewhat higher latency.
