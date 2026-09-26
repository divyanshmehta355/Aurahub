"use client";

import React, { useEffect, useRef } from 'react';
import useSWRInfinite from 'swr/infinite';
import { fetcher } from '@/lib/fetcher';
import ShortPlayer from '@/components/ShortPlayer';

const ShortsFeedClient = () => {
    const getKey = (pageIndex, previousPageData) => {
        if (previousPageData && (!previousPageData.videos?.length || previousPageData.videos.length < 5)) return null;
        return `/videos?sort=trending&type=short&page=${pageIndex + 1}&limit=5`;
    };

    const { data, error, isLoading, isValidating, size, setSize } = useSWRInfinite(
        getKey,
        fetcher,
        {
            revalidateFirstPage: false,
            revalidateAll: false,
            revalidateOnFocus: false,
            revalidateIfStale: false,
            persistSize: false,
        }
    );

    const videos = data ? data.flatMap(page => page.videos || []) : [];
    const isEmpty = data?.[0]?.videos?.length === 0;
    const isReachingEnd = isEmpty || Boolean(data && data[data.length - 1]?.videos?.length < 5);

    const containerRef = useRef(null);
    const isFetchingRef = useRef(false);

    useEffect(() => {
        if (isValidating || isLoading) {
            isFetchingRef.current = true;
        } else {
            const timer = setTimeout(() => {
                isFetchingRef.current = false;
            }, 400);
            return () => clearTimeout(timer);
        }
    }, [isValidating, isLoading]);

    // Keyboard navigation (Arrow keys / J / K)
    useEffect(() => {
        const handleKeyDown = (e) => {
            const container = containerRef.current;
            if (!container) return;

            // Don't trigger if user is typing in an input
            if (["INPUT", "TEXTAREA"].includes(document.activeElement?.tagName)) return;

            if (e.key === "ArrowDown" || e.key === "j" || e.key === "PageDown") {
                e.preventDefault();
                container.scrollBy({ top: container.clientHeight, behavior: "smooth" });
            } else if (e.key === "ArrowUp" || e.key === "k" || e.key === "PageUp") {
                e.preventDefault();
                container.scrollBy({ top: -container.clientHeight, behavior: "smooth" });
            }
        };

        window.addEventListener("keydown", handleKeyDown);
        return () => window.removeEventListener("keydown", handleKeyDown);
    }, []);

    // Infinite scroll handler
    useEffect(() => {
        const container = containerRef.current;
        if (!container) return;

        const handleScroll = () => {
            if (container.scrollTop + container.clientHeight >= container.scrollHeight - 200) {
                if (!isReachingEnd && !isValidating && !isLoading && !isFetchingRef.current) {
                    isFetchingRef.current = true;
                    setSize((prev) => prev + 1);
                }
            }
        };

        container.addEventListener('scroll', handleScroll);
        return () => container.removeEventListener('scroll', handleScroll);
    }, [isReachingEnd, isValidating, isLoading, setSize]);

    return (
        <div 
            ref={containerRef}
            className="h-[calc(100vh-80px)] w-full bg-black overflow-y-scroll snap-y snap-mandatory scroll-smooth scrollbar-hide"
        >
            {videos.map((video, index) => (
                <div key={video._id} className="h-full w-full snap-start flex justify-center items-center">
                    <ShortPlayer video={video} />
                </div>
            ))}
            
            {isLoading && (
                <div className="h-full w-full snap-start flex justify-center items-center text-white">
                    <div className="animate-pulse">Loading more shorts...</div>
                </div>
            )}
            
            {isEmpty && !isLoading && (
                <div className="h-full w-full snap-start flex justify-center items-center text-white">
                    <p>No shorts available yet.</p>
                </div>
            )}
        </div>
    );
};

export default ShortsFeedClient;
