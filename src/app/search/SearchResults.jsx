"use client";

import { useSearchParams } from "next/navigation";
import Link from "next/link";
import useSWR from "swr";
import { fetcher } from "@/lib/fetcher";
import { motion } from "framer-motion";
import VideoCard from "@/components/VideoCard";
import VideoCardSkeleton from "@/components/VideoCardSkeleton";
import { FaMagic } from "react-icons/fa";

const SearchResults = () => {
  const searchParams = useSearchParams();
  const query = searchParams.get("q");

  const { data: rawData, error, isLoading } = useSWR(
    query ? `/videos/search?q=${encodeURIComponent(query)}` : null,
    fetcher
  );

  const videos = Array.isArray(rawData) ? rawData : rawData?.videos || [];
  const didYouMean = rawData?.didYouMean;

  return (
    <motion.main
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.4 }}
      className="container mx-auto px-4 sm:px-6 py-8"
    >
      <div className="mb-6">
        <h2 className="text-2xl sm:text-3xl font-extrabold text-gray-900 dark:text-white font-display tracking-tight">
          Search Results for:{" "}
          <span className="text-indigo-600 dark:text-indigo-400">"{query}"</span>
        </h2>

        {/* AI Did You Mean Banner */}
        {didYouMean && (
          <div className="mt-4 p-4 rounded-2xl bg-indigo-50/80 dark:bg-indigo-950/40 border border-indigo-200/80 dark:border-indigo-800/60 flex items-center gap-2.5 text-sm text-gray-800 dark:text-gray-200">
            <FaMagic className="text-indigo-600 dark:text-indigo-400 flex-shrink-0" />
            <span>Did you mean:</span>
            <Link
              href={`/search?q=${encodeURIComponent(didYouMean)}`}
              className="font-bold text-indigo-600 dark:text-indigo-400 hover:underline hover:text-indigo-700 transition-colors"
            >
              {didYouMean}
            </Link>
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
        {isLoading ? (
          Array.from({ length: 8 }).map((_, index) => (
            <VideoCardSkeleton key={index} />
          ))
        ) : videos.length > 0 ? (
          videos.map((video, index) => (
            <motion.div
              key={video._id}
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ duration: 0.3, delay: index * 0.04 }}
            >
              <VideoCard video={video} />
            </motion.div>
          ))
        ) : (
          <div className="col-span-full text-center p-12 border border-gray-200 dark:border-slate-800 bg-white dark:bg-slate-900 rounded-3xl shadow-sm">
            <p className="text-gray-500 dark:text-gray-400 text-lg font-medium">
              No videos found matching your search.
            </p>
            {didYouMean && (
              <p className="text-sm text-gray-600 dark:text-gray-400 mt-2">
                Try searching for{" "}
                <Link
                  href={`/search?q=${encodeURIComponent(didYouMean)}`}
                  className="font-bold text-indigo-600 dark:text-indigo-400 hover:underline"
                >
                  {didYouMean}
                </Link>
              </p>
            )}
          </div>
        )}
      </div>
    </motion.main>
  );
};

export default SearchResults;