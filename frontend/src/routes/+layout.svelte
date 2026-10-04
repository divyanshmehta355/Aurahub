<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';
	import { Sun, Moon, Search, User, LogOut, Home, PlaySquare, Clock, ListVideo, Menu, PlusCircle, Bird, Smartphone } from 'lucide-svelte';
	import { fetchApi } from '#lib/api';
	import { userState, setUser, clearUser } from '#lib/user.svelte';

	let { children } = $props();
	let isDark = $state(false);
	let isSidebarOpen = $state(false);

	onMount(async () => {
		if (
			localStorage.theme === 'dark' ||
			(!('theme' in localStorage) && window.matchMedia('(prefers-color-scheme: dark)').matches)
		) {
			isDark = true;
			document.documentElement.classList.add('dark');
		} else {
			isDark = false;
			document.documentElement.classList.remove('dark');
		}

		try {
			const data = await fetchApi('/auth/me');
			setUser(data);
		} catch (err) {
			clearUser();
		}
	});

	function toggleTheme() {
		isDark = !isDark;
		if (isDark) {
			document.documentElement.classList.add('dark');
			localStorage.theme = 'dark';
		} else {
			document.documentElement.classList.remove('dark');
			localStorage.theme = 'light';
		}
	}

	function toggleSidebar() {
		isSidebarOpen = !isSidebarOpen;
	}

	async function handleLogout() {
		try {
			await fetchApi('/auth/logout', { method: 'POST' });
			clearUser();
			window.location.href = '/login';
		} catch (err) {
			console.error(err);
		}
	}
</script>

<div class="flex flex-col min-h-screen">
	<!-- Header -->
	<header class="sticky top-0 z-50 w-full border-b bg-background/80 backdrop-blur">
		<div class="flex h-16 items-center justify-between px-4 sm:px-6">
			<!-- Logo -->
			<div class="flex items-center gap-4">
				<button onclick={toggleSidebar} class="p-2 -ml-2 rounded-full hover:bg-muted transition-colors lg:hidden" aria-label="Toggle menu">
					<Menu class="h-5 w-5" />
				</button>
				<a href="/" class="flex items-center gap-2">
					<div class="bg-primary text-primary-foreground p-1.5 rounded-lg">
						<Bird class="w-5 h-5" />
					</div>
					<span class="text-xl font-bold tracking-tight hidden sm:block">Aurahub</span>
				</a>
			</div>

			<!-- Search (Desktop) -->
			<div class="hidden md:flex flex-1 max-w-xl mx-8">
				<form action="/search" method="GET" class="relative w-full">
					<Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
					<input 
						type="search" 
						name="q"
						placeholder="Search videos..." 
						class="w-full bg-muted/50 border border-border rounded-full pl-10 pr-4 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/20 transition-all"
					/>
				</form>
			</div>

			<!-- Actions -->
			<div class="flex items-center gap-2 sm:gap-4">
				<a href="/upload" class="hidden sm:flex items-center gap-2 p-2 px-3 rounded-full hover:bg-muted transition-colors font-medium text-sm border">
					<PlusCircle class="h-4 w-4" />
					Upload
				</a>

				<button onclick={toggleTheme} class="p-2 rounded-full hover:bg-muted transition-colors" aria-label="Toggle theme">
					{#if isDark}
						<Sun class="h-5 w-5" />
					{:else}
						<Moon class="h-5 w-5" />
					{/if}
				</button>
				
				{#if userState.isLoaded}
					{#if userState.user}
						<div class="flex items-center gap-4 ml-2">
							<a href="/profile/{userState.user.username}" class="font-medium text-sm hover:underline">
								@{userState.user.username}
							</a>
							<button onclick={handleLogout} class="p-2 rounded-full hover:bg-muted transition-colors text-red-500" aria-label="Log out">
								<LogOut class="h-5 w-5" />
							</button>
						</div>
					{:else}
						<a href="/login" class="hidden sm:inline-flex items-center justify-center rounded-full text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 hover:bg-muted h-9 px-4 py-2">
							Sign in
						</a>
						<a href="/signup" class="inline-flex items-center justify-center rounded-full text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:pointer-events-none disabled:opacity-50 bg-primary text-primary-foreground shadow hover:bg-primary/90 h-9 px-4 py-2">
							Sign up
						</a>
					{/if}
				{/if}
			</div>
		</div>
	</header>

	<div class="flex flex-1 overflow-hidden relative">
		<!-- Mobile Sidebar Backdrop -->
		{#if isSidebarOpen}
			<button class="fixed inset-0 bg-background/80 backdrop-blur-sm z-40 lg:hidden w-full h-full cursor-default border-none" aria-label="Close sidebar" onclick={toggleSidebar}></button>
		{/if}

		<!-- Sidebar -->
		<aside class={`w-64 border-r overflow-y-auto shrink-0 bg-background absolute lg:static inset-y-0 left-0 z-40 transform transition-transform duration-200 ease-in-out ${isSidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}`}>
			<nav class="p-4 space-y-2 font-medium">
				<a href="/" onclick={() => isSidebarOpen = false} class="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-muted transition-colors"><Home class="w-5 h-5"/> Home</a>
				<a href="/shorts" onclick={() => isSidebarOpen = false} class="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-muted transition-colors"><Smartphone class="w-5 h-5"/> Shorts</a>
				<a href="/subscriptions" onclick={() => isSidebarOpen = false} class="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-muted transition-colors"><PlaySquare class="w-5 h-5"/> Subscriptions</a>
				
				<div class="my-4 border-t"></div>
				<h3 class="px-3 text-sm font-semibold text-muted-foreground mb-2">You</h3>
				<a href="/history" onclick={() => isSidebarOpen = false} class="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-muted transition-colors"><Clock class="w-5 h-5"/> History</a>
				<a href="/watch-later" onclick={() => isSidebarOpen = false} class="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-muted transition-colors"><Clock class="w-5 h-5"/> Watch Later</a>
				<a href="/my-playlists" onclick={() => isSidebarOpen = false} class="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-muted transition-colors"><ListVideo class="w-5 h-5"/> Playlists</a>
			</nav>
		</aside>

		<!-- Main Content -->
		<main class="flex-1 overflow-y-auto">
			<div class="container mx-auto px-4 sm:px-6 py-6 md:py-8 max-w-[1600px]">
				{@render children()}
			</div>
		</main>
	</div>
</div>
