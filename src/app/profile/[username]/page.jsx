"use client";

import React, { useState, useEffect } from 'react';
import { useParams } from 'next/navigation';
import { useSession } from 'next-auth/react';
import Link from 'next/link';
import useSWR from 'swr';
import { fetcher } from '@/lib/fetcher';
import API from '@/lib/api';
import { motion } from 'framer-motion';
import { toast } from 'react-toastify';
import VideoCard from '@/components/VideoCard';
import VideoCardSkeleton from '@/components/VideoCardSkeleton';

const ProfilePage = () => {
    const params = useParams();
    const username = params.username;
    const { data: session, status } = useSession();
    const isAuthenticated = status === "authenticated";
    const currentUser = session?.user;
    
    const { data: profile, error, mutate, isLoading } = useSWR(
        username ? `/users/${username}` : null,
        fetcher
    );

    const handleSubscribe = async () => {
        if (!isAuthenticated) {
            toast.warn("Please log in to subscribe.");
            return;
        }

        const originalProfile = profile;
        const newIsSubscribed = !profile.user.isSubscribed;
        const newSubscriberCount = newIsSubscribed
            ? profile.user.subscriberCount + 1
            : profile.user.subscriberCount - 1;

        mutate({
            ...profile,
            user: { ...profile.user, isSubscribed: newIsSubscribed, subscriberCount: newSubscriberCount }
        }, false);

        try {
            await API.post(`/users/${profile.user.id}/subscribe`);
        } catch (error) {
            toast.error("An error occurred. Please try again.");
            mutate(originalProfile, false);
        }
    };

    if (status === 'loading' || isLoading) {
        return (
             <main className="container mx-auto px-6 py-8 animate-pulse">
                <div className="mb-8">
                    <div className="h-10 bg-gray-300 rounded w-1/3 mb-4"></div>
                    <div className="h-6 bg-gray-300 rounded w-1/4"></div>
                </div>
                <div className="h-8 bg-gray-300 rounded w-1/4 mb-6"></div>
                <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
                    {Array.from({ length: 4 }).map((_, index) => <VideoCardSkeleton key={index} />)}
                </div>
            </main>
        );
    }

    if (error || !profile) {
        return <div className="text-center p-12 text-rose-500 dark:text-rose-400 font-semibold">{error?.message || "User not found."}</div>;
    }
    
    const isOwnProfile = currentUser?.id === profile.user.id;

    return (
        <motion.main 
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.4 }}
            className="container mx-auto px-6 py-8"
        >
            <div className="mb-10 flex flex-col md:flex-row md:justify-between md:items-center gap-6 bg-white dark:bg-slate-900 p-8 rounded-2xl shadow-sm border border-gray-100 dark:border-slate-800 transition-colors duration-300">
                <div>
                    <h1 className="text-4xl font-bold text-gray-900 dark:text-white">{profile.user.username}</h1>
                    <p className="text-gray-600 dark:text-gray-400 mt-2 font-medium">
                        {profile.user.subscriberCount} subscribers • {profile.videos.length} videos • Joined on {new Date(profile.user.joined).toLocaleDateString()}
                    </p>
                </div>
                {isAuthenticated && !isOwnProfile && (
                    <button
                        onClick={handleSubscribe}
                        className={`px-8 py-3 font-semibold rounded-xl transition-all shadow-sm ${
                            profile.user.isSubscribed
                                ? 'bg-gray-100 dark:bg-slate-800 text-gray-800 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-slate-700'
                                : 'bg-gray-900 dark:bg-white text-white dark:text-gray-900 hover:bg-gray-800 dark:hover:bg-gray-100 hover:shadow-md'
                        }`}
                    >
                        {profile.user.isSubscribed ? 'Subscribed' : 'Subscribe'}
                    </button>
                )}
            </div>
            
            <h2 className="text-2xl font-bold text-gray-900 dark:text-white mb-6 border-t border-gray-200 dark:border-slate-800 pt-8">Uploads</h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
                {profile.videos.length > 0 ? (
                    profile.videos.map((video, index) => (
                        <motion.div
                            key={video._id}
                            initial={{ opacity: 0, scale: 0.9 }}
                            animate={{ opacity: 1, scale: 1 }}
                            transition={{ duration: 0.3, delay: index * 0.05 }}
                        >
                            <VideoCard video={video} />
                        </motion.div>
                    ))
                ) : (
                    <div className="col-span-full text-center p-12 border border-gray-200 dark:border-slate-800 bg-white dark:bg-slate-900 rounded-2xl shadow-sm">
                        <p className="text-gray-500 dark:text-gray-400 font-medium">This user hasn't uploaded any videos yet.</p>
                    </div>
                )}
            </div>
        </motion.main>
    );
};

export default ProfilePage;