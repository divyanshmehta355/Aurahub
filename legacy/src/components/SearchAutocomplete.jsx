"use client";

import React, { useState, useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import Image from "next/image";
import { FaSearch } from "react-icons/fa";
import { getAvatarUrl } from "@/lib/identicon";
import { getFallbackThumbnailUrl } from "@/lib/thumbnailSvg";
import { MdClose } from "react-icons/md";
import { useAppStore } from "@/store/useAppStore";
import { useDebounce } from "@/hooks/useDebounce";
import { motion, AnimatePresence } from "framer-motion";

import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Card } from "@/components/ui/card";

const SearchAutocomplete = ({ className }) => {
  const router = useRouter();
  const setMobileMenuOpen = useAppStore((state) => state.setMobileMenuOpen);
  const searchQuery = useAppStore((state) => state.searchQuery);
  const setSearchQuery = useAppStore((state) => state.setSearchQuery);
  
  const [results, setResults] = useState({ videos: [], users: [] });
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  
  const debouncedQuery = useDebounce(searchQuery, 300);
  const dropdownRef = useRef(null);

  useEffect(() => {
    const handleClickOutside = (event) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
        setIsDropdownOpen(false);
      }
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  useEffect(() => {
    const fetchResults = async () => {
      if (debouncedQuery.trim().length < 2) {
        setResults({ videos: [], users: [] });
        setIsDropdownOpen(false);
        return;
      }
      setIsLoading(true);
      try {
        const res = await fetch(`/api/search/autocomplete?q=${encodeURIComponent(debouncedQuery)}`);
        const data = await res.json();
        setResults(data);
        if (data.videos.length > 0 || data.users.length > 0) {
          setIsDropdownOpen(true);
        } else {
          setIsDropdownOpen(false);
        }
      } catch (error) {
        console.error("Failed to fetch autocomplete", error);
      } finally {
        setIsLoading(false);
      }
    };

    fetchResults();
  }, [debouncedQuery]);

  const handleSearchSubmit = () => {
    if (searchQuery.trim() !== "") {
      router.push(`/search?q=${searchQuery.trim()}`);
      setMobileMenuOpen(false);
      setIsDropdownOpen(false);
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === "Enter") {
      handleSearchSubmit();
    }
  };

  const handleClear = () => {
    setSearchQuery("");
    setIsDropdownOpen(false);
    setResults({ videos: [], users: [] });
  };

  const navigateAndClose = (path) => {
    router.push(path);
    setIsDropdownOpen(false);
    setMobileMenuOpen(false);
  };

  return (
    <div className={`relative ${className || ""}`} ref={dropdownRef}>
      <Input
        type="text"
        className="w-full rounded-full pl-4 pr-16 bg-muted/50 border-transparent focus-visible:ring-ring h-10 transition-all"
        placeholder="Search..."
        value={searchQuery}
        onChange={(e) => setSearchQuery(e.target.value)}
        onKeyDown={handleKeyDown}
        onFocus={() => {
          if (results.videos.length > 0 || results.users.length > 0) {
            setIsDropdownOpen(true);
          }
        }}
      />
      <div className="absolute inset-y-0 right-0 flex items-center pr-1.5">
        {searchQuery && (
          <Button
            variant="ghost"
            size="icon"
            onClick={handleClear}
            className="h-7 w-7 rounded-full text-muted-foreground hover:text-foreground hover:bg-muted"
          >
            <MdClose size={16} />
          </Button>
        )}
        <Button
          variant="ghost"
          size="icon"
          onClick={handleSearchSubmit}
          className="h-7 w-7 rounded-full text-muted-foreground hover:text-foreground hover:bg-muted ml-0.5"
        >
          <FaSearch size={14} />
        </Button>
      </div>

      <AnimatePresence>
        {isDropdownOpen && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -10 }}
            transition={{ duration: 0.2 }}
            className="absolute top-full mt-2 w-full z-50"
          >
            <Card className="overflow-hidden shadow-2xl border-border">
              {isLoading && (
                <div className="p-4 text-center text-sm text-muted-foreground">
                  Searching...
                </div>
              )}
              
              {!isLoading && results.users.length > 0 && (
                <div className="border-b border-border">
                  <div className="px-4 py-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider bg-muted/50">
                    Channels
                  </div>
                  {results.users.map((user) => (
                    <div
                      key={user._id}
                      onClick={() => navigateAndClose(`/profile/${user.username}`)}
                      className="flex items-center px-4 py-3 cursor-pointer hover:bg-muted/50 transition-colors"
                    >
                      <Avatar className="h-8 w-8 mr-3">
                        <AvatarImage src={getAvatarUrl(user.username || user.name, user.image || user.avatar)} alt={user.name} />
                        <AvatarFallback>{user.name?.charAt(0) || "U"}</AvatarFallback>
                      </Avatar>
                      <div>
                        <p className="text-sm font-medium text-foreground">
                          {user.name}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          @{user.username}
                        </p>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {!isLoading && results.videos.length > 0 && (
                <div>
                  <div className="px-4 py-2 text-xs font-semibold text-muted-foreground uppercase tracking-wider bg-muted/50">
                    Videos
                  </div>
                  {results.videos.map((video) => (
                    <div
                      key={video._id}
                      onClick={() => navigateAndClose(`/video/${video._id}`)}
                      className="flex items-center px-4 py-3 cursor-pointer hover:bg-muted/50 transition-colors"
                    >
                      <div className="w-16 h-9 relative rounded overflow-hidden mr-3 flex-shrink-0 bg-muted">
                        <Image
                          src={video.thumbnailUrl || getFallbackThumbnailUrl(video._id, video.title, video.category)}
                          alt={video.title}
                          fill
                          unoptimized
                          sizes="64px"
                          className="object-cover"
                        />
                      </div>
                      <p className="text-sm font-medium text-foreground line-clamp-2">
                        {video.title}
                      </p>
                    </div>
                  ))}
                </div>
              )}
              <div 
                className="px-4 py-3 text-sm text-center text-foreground font-medium hover:bg-muted cursor-pointer border-t border-border transition-colors"
                onClick={handleSearchSubmit}
              >
                See all results for "{searchQuery}"
              </div>
            </Card>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
};

export default SearchAutocomplete;
