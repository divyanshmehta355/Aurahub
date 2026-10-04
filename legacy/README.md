<div align="center">
  <h1>Aurahub 🎬</h1>
  <p>A highly-optimized, premium video-sharing platform built with <strong>Next.js 15</strong>, <strong>React 19</strong>, and <strong>Tailwind CSS v4</strong>.</p>
</div>

Aurahub is designed from the ground up for extreme performance and scalability. It features a custom-built, enterprise-grade L1/L2 tiered caching layer, typo-tolerant MongoDB Atlas fuzzy search, and a cinematic front-end UI. 

Whether you are a creator managing uploads or a viewer bingeing content, Aurahub provides a flawless, lag-free experience.

---

## ⚡ Core Architecture & Performance
Aurahub is built to handle heavy traffic and database loads smoothly.

- **Tiered L1/L2 Redis Caching:** Combines a `Map`-based True LRU memory cache (L1) with a Redis cluster (L2). This hybrid approach eliminates network socket latency entirely for hot requests, resolving reads in `<0.1ms`.
- **Thundering Herd Protection:** Concurrent requests for the same cache-missed video are coalesced in-flight, preventing database stampedes during sudden traffic spikes.
- **O(1) Tag-Based Invalidation:** Grouped Redis Sets guarantee instant cache clearing across specific users, videos, and feeds without slow `SCAN` or `KEYS *` operations.
- **Redis Circuit Breakers:** Automatically detects Redis connection failures or max-client exhaustion and bypasses the cache to keep the app online.
- **Typo-Tolerant Atlas Search:** Powered by MongoDB Atlas `$search`, search queries feature native `fuzzy` matching (e.g. typing "mia kalifa" seamlessly finds "Mia Khalifa") using index-level optimizations.

## ✨ Key Features

### For Viewers
* **Immersive Video Player**: Beautiful viewing experience with likes, comments, and instantaneous related-video suggestions.
* **Typo-Tolerant Discovery**: Instant search autocomplete and categorical filtering powered by Atlas Search.
* **Subscriptions Feed**: A dedicated feed to keep up with your favorite creators.
* **Custom Playlists**: Create public or private playlists to curate your favorite content.
* **Watch Later & History**: Automatically track what you watch and save videos for later.

### For Creators
* **Creator Studio / Dashboard**: A professional tabbed dashboard featuring interactive lifetime analytics, 30-day channel growth charts, and video performance comparisons.
* **Content Manager**: Edit video metadata, upload custom thumbnails (via Freeimage API), and toggle video visibility (Public/Unlisted/Private).
* **Public Profiles**: Highly customizable creator profiles featuring a 16:9 cinematic channel banner and custom bios.

### Premium UI/UX
* **Dark Mode First**: Beautiful, class-based dark mode implementation with seamless theme toggling.
* **Cinematic Typography**: Uses `Outfit` for striking headers and `Inter` for highly legible body copy.
* **Micro-Animations**: Fluid transitions, hover effects, and spring animations powered by `framer-motion`.

---

## 🛠 Tech Stack

- **Framework**: Next.js 15 (App Router)
- **Library**: React 19
- **Styling**: Tailwind CSS v4
- **Database**: MongoDB + Mongoose + MongoDB Atlas Search
- **Caching**: Custom L1/L2 LRU Cache via Redis (`redis` client)
- **Authentication**: NextAuth.js (Google, GitHub, Credentials)
- **Data Fetching**: SWR (stale-while-revalidate) + Background Preloading
- **Charts**: Chart.js (`react-chartjs-2`)
- **Animations**: Framer Motion
- **State Management**: Zustand

---

## 🚀 Getting Started

### Prerequisites
You need **Node.js**, a **MongoDB Atlas Cluster** (with a Search index named `"default"` on the videos collection), and a **Redis Instance**.

### 1. Clone the repository
```bash
git clone https://github.com/divyanshmehta355/aurahub.git
cd aurahub
```

### 2. Install dependencies
```bash
npm install
```

### 3. Environment Variables
Create a `.env.local` file in the root directory:

```env
# Core
NODE_ENV=development
MONGO_URI=your_mongodb_atlas_connection_string
REDIS_URL=your_redis_connection_string

# Authentication
NEXTAUTH_SECRET=your_nextauth_secret
NEXTAUTH_URL=http://localhost:3000
JWT_SECRET=your_jwt_secret

# OAuth (Optional)
GITHUB_CLIENT_ID=your_github_client_id
GITHUB_CLIENT_SECRET=your_github_client_secret
GOOGLE_CLIENT_ID=your_google_client_id
GOOGLE_CLIENT_SECRET=your_google_client_secret

# External APIs
FREEIMAGE_API_KEY=your_freeimage_host_key
```

*(Note: Environment variables are strictly validated and tamper-proofed at runtime via `src/env.mjs` and `zod`.)*

### 4. Run the development server
```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

---

## 📂 Project Structure

* `/src/app`: Next.js App Router pages and API routes.
* `/src/components`: Reusable UI components.
* `/src/models`: Mongoose database schemas.
* `/src/lib`: Core infrastructure (Redis client, fetcher, dbConnect).
* `/load-tests`: k6 performance testing suite (deprecated/removed in latest refactor).

## 🤝 Contributing
Contributions, issues and feature requests are welcome! Feel free to check the issues page.

## 📝 License
This project is open-source and available under the [MIT License](LICENSE).
