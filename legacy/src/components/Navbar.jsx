"use client";

import React from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useSession, signOut } from "next-auth/react";
import { getAvatarUrl } from "@/lib/identicon";
import NotificationsPanel from "./NotificationsPanel";
import ThemeToggle from "./ThemeToggle";
import SearchAutocomplete from "./SearchAutocomplete";
import { useAppStore } from "@/store/useAppStore";
import {
  FaUserEdit,
  FaHistory,
  FaHourglassStart,
} from "react-icons/fa";
import { MdDashboard, MdSubscriptions, MdPlaylistPlay, MdLogout, MdAmpStories, MdMenu, MdClose } from "react-icons/md";

import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

const Navbar = () => {
  const { data: session, status } = useSession();
  const isAuthenticated = status === "authenticated";
  const user = session?.user;

  const router = useRouter();
  const isMobileMenuOpen = useAppStore((state) => state.isMobileMenuOpen);
  const setMobileMenuOpen = useAppStore((state) => state.setMobileMenuOpen);
  const toggleMobileMenu = useAppStore((state) => state.toggleMobileMenu);

  const handleLogout = () => {
    signOut({ callbackUrl: '/login' });
  };

  return (
    <nav className="sticky top-0 z-50 backdrop-blur-md bg-background/80 border-b border-border shadow-sm transition-all duration-300">
      <div className="container mx-auto px-4 sm:px-6 py-3">
        <div className="flex justify-between items-center">
          <Link
            href="/"
            className="font-display text-3xl tracking-tighter font-extrabold text-foreground hover:text-foreground/80 transition-all duration-300"
            onClick={() => setMobileMenuOpen(false)}
          >
            Aurahub
          </Link>

          {/* Desktop Menu */}
          <div className="hidden md:flex items-center space-x-4">
            <SearchAutocomplete 
              className="w-72 lg:w-96" 
              inputClassName="w-full bg-muted border-2 border-transparent focus:border-ring rounded-full py-2 pl-4 pr-12 focus:outline-none transition-all duration-300 text-foreground placeholder-muted-foreground" 
            />

            <ThemeToggle />

            {isAuthenticated ? (
              <>
                <Button asChild className="rounded-full shadow-md hover:shadow-lg transition-all duration-300">
                  <Link href="/upload">Upload</Link>
                </Button>
                
                <NotificationsPanel />
                
                <DropdownMenu>
                  <DropdownMenuTrigger className="flex items-center justify-center rounded-full hover:ring-2 hover:ring-ring transition-all duration-300 focus:outline-none cursor-pointer border-none outline-none">
                      <Avatar className="h-9 w-9">
                        <AvatarImage src={getAvatarUrl(user?.name, user?.image)} alt={user?.name || "User avatar"} />
                        <AvatarFallback>{user?.name?.charAt(0) || "U"}</AvatarFallback>
                      </Avatar>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" className="w-64 rounded-xl p-2 animate-in fade-in slide-in-from-top-2">
                    <div className="px-2 py-2.5">
                      <p className="text-sm font-bold truncate">{user?.name}</p>
                      <p className="text-xs text-muted-foreground truncate">{user?.email}</p>
                    </div>
                    <DropdownMenuSeparator />
                    {[
                      { name: "Shorts", icon: MdAmpStories, href: "/shorts" },
                      { name: "My Profile", icon: FaUserEdit, href: "/my-profile" },
                      { name: "Dashboard", icon: MdDashboard, href: "/dashboard" },
                      { name: "Watch Later", icon: FaHourglassStart, href: "/watch-later" },
                      { name: "My Playlists", icon: MdPlaylistPlay, href: "/my-playlists" },
                      { name: "History", icon: FaHistory, href: "/history" },
                      { name: "Subscriptions", icon: MdSubscriptions, href: "/subscriptions" },
                    ].map((item) => (
                      <DropdownMenuItem key={item.name} asChild className="cursor-pointer rounded-lg">
                        <Link href={item.href} className="flex items-center w-full">
                          <item.icon className="mr-3 h-4 w-4 opacity-70" />
                          <span>{item.name}</span>
                        </Link>
                      </DropdownMenuItem>
                    ))}
                    <DropdownMenuSeparator />
                    <DropdownMenuItem onClick={handleLogout} className="cursor-pointer rounded-lg text-destructive focus:bg-destructive/10 focus:text-destructive flex items-center w-full">
                        <MdLogout className="mr-3 h-4 w-4 opacity-80" />
                        <span>Logout</span>
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </>
            ) : (
              <>
                <Button variant="ghost" asChild className="font-medium transition-colors duration-300">
                  <Link href="/login">Login</Link>
                </Button>
                <Button asChild className="rounded-full shadow-md hover:shadow-lg transition-all duration-300">
                  <Link href="/register">Sign Up</Link>
                </Button>
              </>
            )}
          </div>

          {/* Mobile Menu Button */}
          <div className="md:hidden flex items-center gap-3">
            <ThemeToggle />
            <Button
              variant="ghost"
              size="icon"
              onClick={toggleMobileMenu}
              aria-label="Toggle menu"
              className="rounded-full"
            >
              {isMobileMenuOpen ? <MdClose className="h-6 w-6" /> : <MdMenu className="h-6 w-6" />}
            </Button>
          </div>
        </div>

        {/* Mobile Menu */}
        {isMobileMenuOpen && (
          <div className="md:hidden mt-4 pt-4 border-t border-border animate-in slide-in-from-top-4 duration-300">
            <SearchAutocomplete 
              className="mb-6"
              inputClassName="w-full bg-muted border-2 border-transparent focus:border-ring rounded-full py-3 pl-4 pr-12 focus:outline-none transition-all text-foreground placeholder-muted-foreground" 
            />
            
            <div className="flex flex-col space-y-1">
              {isAuthenticated ? (
                <>
                  <div className="flex justify-between items-center p-3 mb-2 bg-muted rounded-xl">
                    <span className="font-semibold text-foreground">
                      Welcome, {user?.name}!
                    </span>
                    <NotificationsPanel />
                  </div>
                  {[
                    { name: "Shorts", href: "/shorts" },
                    { name: "Dashboard", href: "/dashboard" },
                    { name: "My Playlists", href: "/my-playlists" },
                    { name: "Subscriptions", href: "/subscriptions" },
                    { name: "Upload Video", href: "/upload" }
                  ].map((item) => (
                    <Link
                      key={item.name}
                      href={item.href}
                      onClick={() => setMobileMenuOpen(false)}
                      className="px-4 py-3 text-muted-foreground font-medium rounded-xl hover:bg-muted hover:text-foreground transition-colors"
                    >
                      {item.name}
                    </Link>
                  ))}
                  <button
                    onClick={handleLogout}
                    className="text-left px-4 py-3 text-destructive font-bold rounded-xl hover:bg-destructive/10 mt-4 transition-colors"
                  >
                    Logout
                  </button>
                </>
              ) : (
                <div className="grid grid-cols-2 gap-3 mt-4">
                  <Button variant="secondary" asChild className="rounded-xl h-12 text-md font-bold">
                    <Link href="/login" onClick={() => setMobileMenuOpen(false)}>Login</Link>
                  </Button>
                  <Button asChild className="rounded-xl h-12 text-md font-bold shadow-sm">
                    <Link href="/register" onClick={() => setMobileMenuOpen(false)}>Sign Up</Link>
                  </Button>
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </nav>
  );
};

export default Navbar;