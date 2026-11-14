 # ViralForge — Frontend Integration Guide: Location, Merchant, Shop & Product Features

This guide details all backend endpoints, request payloads, response structures, and step-by-step UI/UX flows required for the frontend to integrate with the new Location, Merchant, Shop, Product, and Video Connection architecture.

---

## 1. Authentication & Base URLs

* **Base URL**: `http://localhost:8081` (or production domain)
* **Auth Header**: `Authorization: Bearer <JWT_TOKEN>` (for all protected endpoints)
* **Auth Endpoints** (existing under `/auth/v1`):
  * `POST /auth/v1/register` — Register account
  * `POST /auth/v1/login` — Login (`{ "email": "...", "password": "..." }`)
  * `GET /auth/v1/` — Fetch current user profile (protected)
  * `POST /auth/v1/reset_code` — Send OTP code
  * `POST /auth/v1/verify_code` — Verify OTP
  * `POST /auth/v1/reset_password` — Reset password

---

## 2. API Endpoints Overview

| Module | Method | Endpoint | Auth | Description |
|---|---|---|---|---|
| **Merchants** | `POST` | `/api/v1/merchants` | Required | Create merchant profile |
| | `GET` | `/api/v1/merchants/:id` | Public | Get merchant profile |
| | `PATCH` | `/api/v1/merchants/:id` | Required | Update merchant profile |
| | `GET` | `/api/v1/merchants/:id/shops` | Public | List all shops of a merchant |
| | `GET` | `/api/v1/merchants/:id/videos` | Public | List published videos of a merchant |
| **Shops** | `POST` | `/api/v1/shops` | Required | Create physical shop (with map coordinates) |
| | `GET` | `/api/v1/shops/:id` | Public | Get shop details (with lat/lng coordinates) |
| | `PATCH` | `/api/v1/shops/:id` | Required | Update shop details / location |
| | `GET` | `/api/v1/shops/:id/products` | Public | Get shop's product catalogue |
| | `GET` | `/api/v1/shops/:id/videos` | Public | Get videos featuring this shop |
| | `POST` | `/api/v1/shops/:id/products` | Required | Add product to shop catalogue (price & stock) |
| | `GET` | `/api/v1/shops/nearby` | Public | Discover nearby verified shops |
| **Products** | `POST` | `/api/v1/products` | Required | Create product in central catalogue |
| | `GET` | `/api/v1/products/:id` | Public | Get central product details |
| | `GET` | `/api/v1/products/:id/shops` | Public | Get shops selling this product (ordered by price) |
| | `PATCH` | `/api/v1/shop-products/:id` | Required | Update shop price, availability, or status |
| **Discovery** | `GET` | `/api/v1/search/products` | Public | Search products by keyword |
| | `GET` | `/api/v1/search/nearby` | Public | PostGIS nearby search for shops carrying a specific product |
| | `GET` | `/api/v1/discover/shops` | Public | Browse verified shops by city/keyword |
| | `GET` | `/api/v1/discover/videos` | Public | Public video feed |
| | `GET` | `/api/v1/feed/videos` | Public | Public video feed |
| **Customer Locations** | `POST` | `/api/v1/locations/saved` | Required | Save customer location (e.g., Home, Work) |
| | `GET` | `/api/v1/locations/saved` | Required | Get user's saved locations |
| | `DELETE` | `/api/v1/locations/saved/:id` | Required | Delete a saved location |
| **Videos** | `POST` | `/v1/upload/initiate` | Required | Get presigned S3 upload URL |
| | `POST` | `/v1/upload/commit` | Required | Commit upload & attach merchant, shop, product |
| | `GET` | `/v1/videos/:v_id` | Public | Full video details + Merchant + Shop + Product |
| | `GET` | `/v1/videos/:v_id/related-shops` | Public | Shops selling product featured in video |
| | `GET` | `/v1/videos/:v_id/related-products` | Public | Products related to this video/shop |

---

## 3. UI/UX Flows & Screen Requirements

### Flow 1: Merchant Onboarding & Shop Creation

```text
[User Settings / Profile]
        |
        v
"Become a Merchant" Button
        |
        v
[Merchant Registration Form] (POST /api/v1/merchants)
        |
        v
[Add Shop Screen with Map Picker] (POST /api/v1/shops)
        |
        v
[Shop Dashboard] -> Add Products to Catalogue (POST /api/v1/shops/:id/products)
```

#### Step 1: Create Merchant Profile
* **Endpoint**: `POST /api/v1/merchants`
* **Request**:
  ```json
  {
    "business_name": "ABC Electronics",
    "business_description": "Authorized retail and repairs for top brands",
    "business_type": "RETAIL" // "RETAIL" | "ONLINE" | "SERVICE"
  }
  ```
* **Response (201 Created)**:
  ```json
  {
    "success": true,
    "data": {
      "id": 1,
      "owner_user_id": 12,
      "business_name": "ABC Electronics",
      "business_description": "Authorized retail and repairs for top brands",
      "business_type": "RETAIL",
      "verification_status": "PENDING"
    },
    "message": "Merchant profile created successfully"
  }
  ```

#### Step 2: Register a Physical Shop with Map Coordinates
* Display an interactive map picker (e.g. Leaflet, Mapbox, or Google Maps) allowing the merchant to place a pin or drag marker to select shop coordinates.
* **Endpoint**: `POST /api/v1/shops`
* **Request**:
  ```json
  {
    "merchant_id": 1,
    "name": "ABC Electronics - Dibrugarh Branch",
    "description": "Flagship store in Dibrugarh",
    "phone": "+91 9876543210",
    "address": "R.K.B. Path, Near Thana Chariali",
    "city": "Dibrugarh",
    "state": "Assam",
    "pincode": "786001",
    "country": "India",
    "latitude": 27.4728,
    "longitude": 94.9120
  }
  ```
* **Response (201 Created)**:
  ```json
  {
    "success": true,
    "data": {
      "id": 10,
      "merchant_id": 1,
      "name": "ABC Electronics - Dibrugarh Branch",
      "latitude": 27.4728,
      "longitude": 94.9120,
      "status": "ACTIVE",
      "verification_status": "PENDING"
    },
    "message": "Shop created successfully"
  }
  ```

---

### Flow 2: Adding Products & Managing Shop Catalogue

Merchants can either pick an existing central product or create a new central product, then assign their shop's price and availability.

#### Step 1: Search Existing Central Products or Create New
* Search: `GET /api/v1/search/products?q=Samsung`
* Or Create: `POST /api/v1/products`
  ```json
  {
    "name": "Samsung Galaxy S25",
    "brand": "Samsung",
    "description": "Flagship 256GB Titanium Gray",
    "specifications": {
      "ram": "12GB",
      "storage": "256GB",
      "color": "Titanium Gray"
    }
  }
  ```

#### Step 2: Add Product to Shop Catalogue
* **Endpoint**: `POST /api/v1/shops/:id/products`
* **Request**:
  ```json
  {
    "product_id": 101,
    "price": 75000,
    "currency": "INR",
    "availability": "AVAILABLE" // "AVAILABLE" | "UNKNOWN" | "OUT_OF_STOCK"
  }
  ```

#### Step 3: Fast Stock / Price Update (Merchant Dashboard)
* **Endpoint**: `PATCH /api/v1/shop-products/:id` (using the listing ID)
* **Request**:
  ```json
  {
    "price": 73500,
    "availability": "OUT_OF_STOCK"
  }
  ```

---

### Flow 3: Merchant Video Upload & Entity Attachment

When uploading a video, provide optional selectors so the merchant can link:
1. **Merchant Profile**
2. **Shop** (filtered by selected merchant)
3. **Product** & **Shop Listing**

#### Step 1: Initiate Upload (Existing)
* `POST /v1/upload/initiate`
  * Body: `{ "filename": "demo_video.mp4" }`
  * Response contains `objectKey` and presigned S3 `url`.
* Frontend uploads binary directly to S3 `url`.

#### Step 2: Commit Video Upload with Entity Relationships
* **Endpoint**: `POST /v1/upload/commit`
* **Request**:
  ```json
  {
    "filename": "demo_video.mp4",
    "fileType": "video/mp4",
    "objectKey": "videos/abc-uuid-demo_video.mp4",
    "merchant_id": 1,
    "shop_id": 10,
    "product_id": 101,
    "shop_product_id": 55
  }
  ```
  *(Note: All four entity fields are optional; creators can post videos without merchant or shop links!)*

---

### Flow 4: Customer Video Player & Rich Information Overlay

When a customer opens or swipes to a video (`GET /v1/videos/:v_id`), the response provides the complete aggregated data tree in a single call.

* **Endpoint**: `GET /v1/videos/:v_id`
* **Response (200 OK)**:
  ```json
  {
    "video": {
      "id": 100,
      "title": "Samsung Galaxy S25 Hands-on Demo",
      "description": "Available now in our Dibrugarh showroom!",
      "playback_url": "https://cdn.example.com/videos/master.m3u8",
      "thumbnail": "https://cdn.example.com/thumbnails/thumb1.jpg",
      "status": "published",
      "duration": 45.2
    },
    "creator": {
      "user_id": 12,
      "name": "Punam",
      "avatar": "https://..."
    },
    "merchant": {
      "id": 1,
      "business_name": "ABC Electronics",
      "business_type": "RETAIL"
    },
    "shop": {
      "id": 10,
      "name": "ABC Electronics - Dibrugarh Branch",
      "address": "R.K.B. Path, Near Thana Chariali",
      "phone": "+91 9876543210",
      "latitude": 27.4728,
      "longitude": 94.9120
    },
    "product": {
      "id": 101,
      "name": "Samsung Galaxy S25",
      "brand": "Samsung",
      "description": "Flagship 256GB Titanium Gray"
    },
    "shop_listing": {
      "id": 55,
      "price": 75000,
      "currency": "INR",
      "availability": "AVAILABLE",
      "stock_updated_at": "2026-11-12T20:15:00Z"
    }
  }
  ```

#### Recommended Video UI Elements:
1. **Bottom Sheet / Card**:
   * Show Product Title: `"Samsung Galaxy S25"`
   * Price Tag: `"₹75,000"` (with status badge: `"In Stock"` or `"Out of Stock"` based on `availability`).
   * Stock freshness indicator: e.g. `"Confirmed 2 hours ago"`.
2. **"View Shop" / "Get Directions" Button**:
   * Tapping opens Google Maps/Apple Maps navigation:
     `https://www.google.com/maps/dir/?api=1&destination=27.4728,94.9120`
   * Or opens the **Shop Profile Page** (`/shops/:id`).
3. **"Other Shops Selling This" Drawer**:
   * Call `GET /v1/videos/:v_id/related-shops` to display a list of all other nearby shops carrying the product and their respective prices.

---

### Flow 5: Nearby Location Discovery & Maps

#### 1. Customer Location Selector
Allow customer to choose their location from:
1. **Browser GPS** (`navigator.geolocation.getCurrentPosition`)
2. **Saved Locations** (`GET /api/v1/locations/saved`)

#### 2. Saving a Location
* `POST /api/v1/locations/saved`
  ```json
  {
    "label": "Home",
    "address": "Borpather, Dibrugarh",
    "city": "Dibrugarh",
    "pincode": "786001",
    "latitude": 27.4800,
    "longitude": 94.9100
  }
  ```

#### 3. Nearby Product Search (Radius Filter: 1 km, 5 km, 10 km)
* **Endpoint**: `GET /api/v1/search/nearby`
* **Query Params**:
  * `latitude`: float (e.g. `27.4728`)
  * `longitude`: float (e.g. `94.9120`)
  * `product_id`: integer (e.g. `101`)
  * `radius`: integer in meters (e.g. `5000` for 5 km, `10000` for 10 km)
* **Response (200 OK)**:
  ```json
  {
    "success": true,
    "data": [
      {
        "shop_id": 10,
        "shop_name": "ABC Electronics Dibrugarh",
        "address": "R.K.B. Path, Near Thana Chariali",
        "price": 73500,
        "availability": "AVAILABLE",
        "distance_meters": 450.2
      },
      {
        "shop_id": 14,
        "shop_name": "City Digital Hub",
        "address": "Mancotta Road",
        "price": 75000,
        "availability": "AVAILABLE",
        "distance_meters": 1820.5
      }
    ]
  }
  ```

#### 4. Nearby Shops Map View
* **Endpoint**: `GET /api/v1/shops/nearby?latitude=27.4728&longitude=94.9120&radius=5000`
* Returns all active verified shops within 5 km, ordered by closest distance.
* Frontend renders shop markers directly on the map using `latitude` and `longitude`.

---

## 4. Frontend Route Structure Recommendations

| Route Path | Component / Page | Purpose |
|---|---|---|
| `/feed` | `VideoFeedPage` | Main short-video discovery feed |
| `/video/:id` | `VideoPlayerModal` | Single video view with Product & Shop overlay |
| `/merchants/:id` | `MerchantProfilePage` | Business bio, all shops, and all videos |
| `/shops/:id` | `ShopDetailsPage` | Shop address, map view, contact info, catalogue |
| `/products/:id` | `ProductDetailsPage` | Central specs, nearby shops comparison table |
| `/explore/nearby` | `NearbyExplorePage` | Map with search bar + radius slider (1km, 5km, 10km) |
| `/dashboard/merchant` | `MerchantDashboardPage` | Manage shops, catalogue, stock, and upload videos |
| `/profile/locations` | `SavedLocationsPage` | Manage saved Home/Work locations |

---

## 5. TypeScript Interfaces

```typescript
// Enums
export type BusinessType = 'RETAIL' | 'ONLINE' | 'SERVICE';
export type VerificationStatus = 'PENDING' | 'VERIFIED' | 'REJECTED';
export type StockAvailability = 'AVAILABLE' | 'UNKNOWN' | 'OUT_OF_STOCK';
export type ShopStatus = 'ACTIVE' | 'INACTIVE';

// Models
export interface MerchantProfile {
  id: number;
  owner_user_id: number;
  business_name: string;
  business_description: string;
  business_type: BusinessType;
  verification_status: VerificationStatus;
  created_at: string;
}

export interface Shop {
  id: number;
  merchant_id: number;
  name: string;
  description: string;
  phone: string;
  address: string;
  city: string;
  state: string;
  pincode: string;
  country: string;
  latitude: number;
  longitude: number;
  status: ShopStatus;
  verification_status: VerificationStatus;
}

export interface Product {
  id: number;
  name: string;
  brand: string;
  description: string;
  specifications?: Record<string, any>;
}

export interface ShopProductListing {
  id: number;
  shop_id: number;
  product_id: number;
  price: number;
  currency: string;
  availability: StockAvailability;
  stock_updated_at: string;
  status: 'ACTIVE' | 'INACTIVE';
  product?: Product;
}

export interface VideoDetailsResponse {
  video: {
    id: number;
    title: string;
    description: string;
    playback_url: string;
    thumbnail: string;
    status: string;
    duration: number;
  };
  creator?: {
    user_id: number;
    name: string;
    avatar: string;
  };
  merchant?: {
    id: number;
    business_name: string;
    business_type: BusinessType;
  };
  shop?: {
    id: number;
    name: string;
    address: string;
    phone: string;
    latitude: number;
    longitude: number;
  };
  product?: Product;
  shop_listing?: {
    id: number;
    price: number;
    currency: string;
    availability: StockAvailability;
    stock_updated_at: string;
  };
}

export interface NearbyShopResult {
  id: number;
  merchant_id: number;
  name: string;
  description: string;
  phone: string;
  address: string;
  city: string;
  state: string;
  pincode: string;
  country: string;
  latitude: number;
  longitude: number;
  verification_status: VerificationStatus;
  distance_meters: number;
}
```
