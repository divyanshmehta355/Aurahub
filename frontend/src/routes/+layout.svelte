<script lang="ts">
	import '../app.css';
	import { onMount, onDestroy } from 'svelte';
	import {
		Sun,
		Moon,
		Search,
		User,
		LogOut,
		Home,
		PlaySquare,
		Clock,
		ListVideo,
		Menu,
		PlusCircle,
		Bird,
		Smartphone,
		Video,
		BarChart,
		Bell,
		ShieldAlert,
		ShieldCheck,
		Shield,
		AlertOctagon,
		Zap
	} from 'lucide-svelte';
	import { fetchApi } from '#lib/api';
	import { userState, setUser, clearUser } from '#lib/user.svelte';
	import { contentModeState, initContentMode, toggleContentMode, setContentMode } from '#lib/consent.svelte';
	import NotificationBell from '#lib/components/NotificationBell.svelte';
	import SearchAutocomplete from '#lib/components/SearchAutocomplete.svelte';
	import AdultConsentModal from '#lib/components/AdultConsentModal.svelte';

	import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';

	let { children } = $props();
	let isDark = $state(false);
	let isSidebarOpen = $state(false);
	let isConsentModalOpen = $state(false);

	let isMobileSearchOpen = $state(false);

	interface SystemStatus {
		status: string;
		maintenance_mode: boolean;
		freeze_uploads: boolean;
		freeze_registration: boolean;
		freeze_comments: boolean;
		timestamp: string;
	}

	let systemStatus = $state<SystemStatus | null>(null);
	let statusInterval: ReturnType<typeof setInterval> | null = null;

	async function fetchSystemStatus() {
		try {
			const res = await fetchApi('/system/status');
			systemStatus = res;
		} catch (e) {
			// Fail-safe: ignore network errors on background poll
		}
	}

	const queryClient = new QueryClient({
		defaultOptions: {
			queries: {
				staleTime: 1000 * 60 * 5, // 5 minutes
				refetchOnWindowFocus: false
			}
		}
	});

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

		initContentMode();
		fetchSystemStatus();
		statusInterval = setInterval(fetchSystemStatus, 15000);

		try {
			const data = await fetchApi('/auth/me');
			setUser(data);
			if (!data?.showAdultContent && contentModeState.isAdult) {
				setContentMode(false);
			}
		} catch (err) {
			clearUser();
			if (contentModeState.isAdult) {
				setContentMode(false);
			}
		}
	});

	onDestroy(() => {
		if (statusInterval) {
			clearInterval(statusInterval);
		}
	});

	$effect(() => {
		if (userState.isLoaded) {
			const canUseAdult = !!userState.user?.showAdultContent;
			if (!canUseAdult && contentModeState.isAdult) {
				setContentMode(false);
			}
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

	function handleToggleContentMode() {
		toggleContentMode(() => {
			isConsentModalOpen = true;
		});
		queryClient.invalidateQueries();
	}

	function onConsentConfirmed() {
		setContentMode(true);
		queryClient.invalidateQueries();
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

<QueryClientProvider client={queryClient}>
	<div class="flex min-h-screen flex-col pb-16 md:pb-0">
		<!-- Emergency Kill Switch Alert Banners -->
		{#if systemStatus?.maintenance_mode}
			<div
				class="relative z-50 flex items-center justify-between border-b border-rose-500/30 bg-gradient-to-r from-rose-950/90 via-amber-950/80 to-rose-950/90 px-4 py-2 text-xs font-semibold text-rose-200 backdrop-blur shadow-md"
			>
				<div class="mx-auto flex flex-wrap items-center justify-center gap-2 text-center">
					<span class="relative flex h-2.5 w-2.5">
						<span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-rose-400 opacity-75"></span>
						<span class="relative inline-flex h-2.5 w-2.5 rounded-full bg-rose-500"></span>
					</span>
					<span class="font-black uppercase tracking-wider text-rose-300">Platform Maintenance Mode:</span>
					<span>Aurahub is in maintenance. Public modifications are temporarily paused.</span>
					{#if userState.user?.isAdmin}
						<span class="rounded-full bg-amber-500/20 px-2 py-0.5 text-[10px] font-black uppercase text-amber-300 border border-amber-500/40">
							Admin Superpower Bypass Active
						</span>
					{/if}
				</div>
			</div>
		{:else if systemStatus?.freeze_uploads}
			<div
				class="relative z-50 flex items-center justify-between border-b border-amber-500/30 bg-amber-950/80 px-4 py-1.5 text-xs font-semibold text-amber-200 backdrop-blur"
			>
				<div class="mx-auto flex flex-wrap items-center justify-center gap-2 text-center">
					<span class="h-2 w-2 rounded-full bg-amber-400 animate-pulse"></span>
					<span class="font-bold uppercase tracking-wider text-amber-300">Notice:</span>
					<span>Video uploads are temporarily frozen for storage optimization. Watching & browsing remain active.</span>
					{#if userState.user?.isAdmin}
						<span class="rounded-full bg-amber-500/30 px-2 py-0.5 text-[10px] font-bold text-amber-200 border border-amber-500/40">
							Admin Bypass
						</span>
					{/if}
				</div>
			</div>
		{/if}

		<!-- Header -->
		<header class="sticky top-0 z-50 w-full border-b bg-background/80 backdrop-blur">
			<div class="flex h-16 items-center justify-between px-4 sm:px-6">
				{#if isMobileSearchOpen}
					<!-- Mobile Search View -->
					<div class="flex w-full items-center gap-2 md:hidden">
						<button
							onclick={() => (isMobileSearchOpen = false)}
							class="hover:bg-muted -ml-2 rounded-full p-2"
							aria-label="Close search"
						>
							<Search class="h-5 w-5 rotate-90" />
							<!-- makeshift back arrow or use ArrowLeft if imported -->
						</button>
						<SearchAutocomplete
							autofocus={true}
							onSearchSubmitted={() => (isMobileSearchOpen = false)}
							class="flex-1"
						/>
					</div>
				{:else}
					<!-- Logo -->
					<div class="flex items-center gap-4">
						<button
							onclick={toggleSidebar}
							class="hover:bg-muted -ml-2 rounded-full p-2 transition-colors lg:hidden"
							aria-label="Toggle menu"
						>
							<Menu class="h-5 w-5" />
						</button>
						<a href="/" class="flex items-center gap-2">
							<div class="text-primary-foreground rounded-lg bg-primary p-1.5">
								<Bird class="h-5 w-5" />
							</div>
							<span class="hidden text-xl font-bold tracking-tight sm:block">Aurahub</span>
						</a>
					</div>

					<!-- Search (Desktop) -->
					<div class="mx-8 hidden max-w-xl flex-1 md:flex">
						<SearchAutocomplete />
					</div>

					<!-- Actions -->
					<div class="flex items-center gap-2 sm:gap-4">
						<!-- Mobile Search Toggle -->
						<button
							onclick={() => (isMobileSearchOpen = true)}
							class="hover:bg-muted rounded-full p-2 md:hidden"
							aria-label="Open search"
						>
							<Search class="h-5 w-5" />
						</button>

						<a
							href="/upload"
							class="hover:bg-muted hidden items-center gap-2 rounded-full border p-2 px-3 text-sm font-medium transition-colors sm:flex"
						>
							<PlusCircle class="h-4 w-4" />
							Upload
						</a>

						<button
							onclick={toggleTheme}
							class="hover:bg-muted rounded-full p-2 transition-colors"
							aria-label="Toggle theme"
						>
							{#if isDark}
								<Sun class="h-5 w-5" />
							{:else}
								<Moon class="h-5 w-5" />
							{/if}
						</button>

						<!-- Content Mode Toggle (General Safe vs 18+ Adult) - Only visible when enabled on /profile -->
						{#if userState.isLoaded && userState.user?.showAdultContent}
							<button
								onclick={handleToggleContentMode}
								class="flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-semibold transition-all duration-200 select-none {contentModeState.isAdult
									? 'border-rose-500/50 bg-rose-500/15 text-rose-500 shadow-sm shadow-rose-500/20 hover:bg-rose-500/25'
									: 'border-border/80 bg-muted/40 text-muted-foreground hover:bg-muted hover:text-foreground'}"
								title={contentModeState.isAdult
									? 'Adult Mode Active (18+) - Click to switch to General Content'
									: 'General Safe Mode - Click to switch to 18+ Content'}
								aria-label="Toggle content mode"
							>
								{#if contentModeState.isAdult}
									<span class="inline-block h-2 w-2 rounded-full bg-rose-500 animate-pulse"></span>
									<span>18+</span>
								{:else}
									<ShieldCheck class="h-3.5 w-3.5 text-emerald-500" />
									<span class="hidden sm:inline">Safe</span>
								{/if}
							</button>
						{/if}

						{#if userState.isLoaded}
							{#if userState.user}
								<NotificationBell />
								<div class="ml-2 flex items-center gap-4">
									{#if userState.user.isAdmin}
										<a
											href="/admin"
											class="hidden items-center gap-1.5 rounded-full border border-amber-500/40 bg-amber-500/15 px-2.5 py-1 text-xs font-bold text-amber-500 transition-colors hover:bg-amber-500/25 sm:flex"
											title="Admin Console"
										>
											<ShieldCheck class="h-3.5 w-3.5" />
											<span>Admin</span>
										</a>
									{/if}
									<a
										href="/dashboard"
										class="hidden rounded bg-primary/10 px-2 py-1 text-xs font-semibold text-primary transition-colors hover:bg-primary/20 sm:block"
									>
										Studio
									</a>
									<a
										href="/profile/{userState.user.username}"
										class="flex items-center gap-2 transition-opacity hover:opacity-80"
									>
										<img
											src={userState.user.avatar ||
												`https://api.dicebear.com/7.x/identicon/svg?seed=${userState.user.username}`}
											alt={userState.user.username}
											class="bg-muted h-8 w-8 rounded-full border border-border"
										/>
										<span class="hidden text-sm font-medium sm:block"
											>@{userState.user.username}</span
										>
									</a>
									<button
										onclick={handleLogout}
										class="hover:bg-muted rounded-full p-2 text-red-500 transition-colors"
										aria-label="Log out"
									>
										<LogOut class="h-5 w-5" />
									</button>
								</div>
							{:else}
								<a
									href="/login"
									class="focus-visible:ring-ring hover:bg-muted hidden h-9 items-center justify-center rounded-full px-4 py-2 text-sm font-medium transition-colors focus-visible:ring-1 focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50 sm:inline-flex"
								>
									Sign in
								</a>
								<a
									href="/signup"
									class="focus-visible:ring-ring text-primary-foreground inline-flex h-9 items-center justify-center rounded-full bg-primary px-4 py-2 text-sm font-medium shadow transition-colors hover:bg-primary/90 focus-visible:ring-1 focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50"
								>
									Sign up
								</a>
							{/if}
						{/if}
					</div>
				{/if}
			</div>
		</header>

		<div class="relative flex flex-1 overflow-hidden">
			<!-- Mobile Sidebar Backdrop -->
			{#if isSidebarOpen}
				<button
					class="fixed inset-0 z-40 h-full w-full cursor-default border-none bg-background/80 backdrop-blur-sm lg:hidden"
					aria-label="Close sidebar"
					onclick={toggleSidebar}
				></button>
			{/if}

			<!-- Sidebar -->
			<aside
				class={`absolute inset-y-0 left-0 z-40 w-64 shrink-0 transform overflow-y-auto border-r bg-background transition-transform duration-200 ease-in-out lg:static ${isSidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}`}
			>
				<nav class="space-y-2 p-4 font-medium">
					<a
						href="/"
						onclick={() => (isSidebarOpen = false)}
						class="hover:bg-muted flex items-center gap-3 rounded-lg px-3 py-2 transition-colors"
						><Home class="h-5 w-5" /> Home</a
					>
					<a
						href="/shorts"
						onclick={() => (isSidebarOpen = false)}
						class="hover:bg-muted flex items-center gap-3 rounded-lg px-3 py-2 transition-colors"
						><Smartphone class="h-5 w-5" /> Shorts</a
					>
					<a
						href="/subscriptions"
						onclick={() => (isSidebarOpen = false)}
						class="hover:bg-muted flex items-center gap-3 rounded-lg px-3 py-2 transition-colors"
						><PlaySquare class="h-5 w-5" /> Subscriptions</a
					>

					<div class="my-4 border-t"></div>
					<h3 class="text-muted-foreground mb-2 px-3 text-sm font-semibold">You</h3>
					<a
						href="/notifications"
						onclick={() => (isSidebarOpen = false)}
						class="hover:bg-muted flex items-center gap-3 rounded-lg px-3 py-2 transition-colors"
						><Bell class="h-5 w-5" /> Notifications</a
					>
					<a
						href="/history"
						onclick={() => (isSidebarOpen = false)}
						class="hover:bg-muted flex items-center gap-3 rounded-lg px-3 py-2 transition-colors"
						><Clock class="h-5 w-5" /> History</a
					>
					<a
						href="/watch-later"
						onclick={() => (isSidebarOpen = false)}
						class="hover:bg-muted flex items-center gap-3 rounded-lg px-3 py-2 transition-colors"
						><Clock class="h-5 w-5" /> Watch Later</a
					>
					<a
						href="/my-playlists"
						onclick={() => (isSidebarOpen = false)}
						class="hover:bg-muted flex items-center gap-3 rounded-lg px-3 py-2 transition-colors"
						><ListVideo class="h-5 w-5" /> Playlists</a
					>

					<div class="my-4 border-t"></div>
					<h3 class="text-muted-foreground mb-2 px-3 text-sm font-semibold">Creator Studio</h3>
					<a
						href="/dashboard"
						onclick={() => (isSidebarOpen = false)}
						class="hover:bg-muted flex items-center gap-3 rounded-lg px-3 py-2 transition-colors"
						><Video class="h-5 w-5" /> Content</a
					>
					<a
						href="/analytics"
						onclick={() => (isSidebarOpen = false)}
						class="hover:bg-muted flex items-center gap-3 rounded-lg px-3 py-2 transition-colors"
						><BarChart class="h-5 w-5" /> Analytics</a
					>

					{#if userState.isLoaded && userState.user?.isAdmin}
						<div class="my-4 border-t border-border/70"></div>
						<h3 class="text-amber-500/90 mb-2 flex items-center gap-1.5 px-3 text-xs font-bold uppercase tracking-wider">
							<ShieldCheck class="h-3.5 w-3.5" /> Administration
						</h3>
						<a
							href="/admin"
							onclick={() => (isSidebarOpen = false)}
							class="text-amber-500 hover:bg-amber-500/10 flex items-center gap-3 rounded-lg px-3 py-2 font-medium transition-colors"
						>
							<Shield class="h-5 w-5" /> Admin Console
						</a>
					{/if}
				</nav>
			</aside>

			<!-- Main Content -->
			<main class="flex-1 overflow-y-auto">
				<div class="container mx-auto max-w-[1600px] px-4 py-6 sm:px-6 md:py-8">
					{@render children()}
				</div>
			</main>
		</div>

		<!-- Mobile Bottom Navigation -->
		<nav
			class="text-muted-foreground pb-safe fixed right-0 bottom-0 left-0 z-50 flex h-16 items-center justify-around border-t bg-background px-2 text-xs font-medium md:hidden"
		>
			<a
				href="/"
				class="hover:text-foreground flex flex-col items-center gap-1 p-2 transition-colors"
			>
				<Home class="h-6 w-6" />
				<span>Home</span>
			</a>
			<a
				href="/shorts"
				class="hover:text-foreground flex flex-col items-center gap-1 p-2 transition-colors"
			>
				<Smartphone class="h-6 w-6" />
				<span>Shorts</span>
			</a>
			<a
				href="/upload"
				class="hover:text-foreground flex flex-col items-center gap-1 p-2 transition-colors"
			>
				<div class="text-primary-foreground -mt-2 rounded-full bg-primary p-1 shadow-lg">
					<PlusCircle class="h-6 w-6" />
				</div>
				<span>Upload</span>
			</a>
			<a
				href="/subscriptions"
				class="hover:text-foreground flex flex-col items-center gap-1 p-2 transition-colors"
			>
				<PlaySquare class="h-6 w-6" />
				<span>Subs</span>
			</a>
			{#if userState.isLoaded && userState.user}
				<a
					href="/profile/{userState.user.username}"
					class="hover:text-foreground flex flex-col items-center gap-1 p-2 transition-colors"
				>
					<img
						src={userState.user.avatar ||
							`https://api.dicebear.com/7.x/identicon/svg?seed=${userState.user.username}`}
						alt={userState.user.username}
						class="bg-muted h-6 w-6 rounded-full border border-border"
					/>
					<span>You</span>
				</a>
			{:else}
				<a
					href="/login"
					class="hover:text-foreground flex flex-col items-center gap-1 p-2 transition-colors"
				>
					<User class="h-6 w-6" />
					<span>Sign in</span>
				</a>
			{/if}
		</nav>
	</div>

	<AdultConsentModal
		bind:isOpen={isConsentModalOpen}
		onConsent={onConsentConfirmed}
	/>
</QueryClientProvider>
