<script lang="ts">
	import '../app.css';
	import { onMount, onDestroy } from 'svelte';
	import { page } from '$app/state';
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
	let currentPath = $derived(page.url.pathname);

	function isActive(href: string) {
		return href === '/' ? currentPath === '/' : currentPath.startsWith(href);
	}

	interface SystemStatus {
		status: string;
		maintenance_mode?: boolean;
		maintenanceMode?: boolean;
		freeze_uploads?: boolean;
		freezeUploads?: boolean;
		freeze_registration?: boolean;
		freezeRegistration?: boolean;
		freeze_comments?: boolean;
		freezeComments?: boolean;
		freeze_streaming?: boolean;
		freezeStreaming?: boolean;
		timestamp?: string;
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
		{:else if systemStatus?.freeze_streaming || systemStatus?.freezeStreaming}
			<div
				class="relative z-50 flex items-center justify-between border-b border-rose-500/40 bg-rose-950/90 px-4 py-2 text-xs font-semibold text-rose-200 backdrop-blur"
			>
				<div class="mx-auto flex flex-wrap items-center justify-center gap-2 text-center">
					<span class="relative flex h-2 w-2">
						<span class="absolute inline-flex h-full w-full animate-ping rounded-full bg-rose-400 opacity-75"></span>
						<span class="relative inline-flex h-2 w-2 rounded-full bg-rose-500"></span>
					</span>
					<span class="font-black uppercase tracking-wider text-rose-300">Streaming Blackout Active:</span>
					<span>Video playback has been temporarily halted by administrators. Browsing, search, and accounts remain active.</span>
					{#if userState.user?.isAdmin}
						<span class="rounded-full bg-amber-500/30 px-2 py-0.5 text-[10px] font-bold text-amber-200 border border-amber-500/40">
							Admin Playback Bypass Active
						</span>
					{/if}
				</div>
			</div>
		{:else if systemStatus?.freeze_uploads || systemStatus?.freezeUploads}
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
		<header class="sticky top-0 z-50 w-full border-b border-border/60 bg-background/85 backdrop-blur-xl transition-colors">
			<div class="flex h-16 items-center justify-between px-4 sm:px-6">
				{#if isMobileSearchOpen}
					<!-- Mobile Search View -->
					<div class="flex w-full items-center gap-2 md:hidden">
						<button
							onclick={() => (isMobileSearchOpen = false)}
							class="hover:bg-secondary/80 -ml-2 rounded-full p-2 text-muted-foreground hover:text-foreground"
							aria-label="Close search"
						>
							<Search class="h-5 w-5 rotate-90" />
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
							class="hover:bg-secondary/80 -ml-2 rounded-full p-2 text-muted-foreground hover:text-foreground transition-colors lg:hidden"
							aria-label="Toggle menu"
						>
							<Menu class="h-5 w-5" />
						</button>
						<a href="/" class="group flex items-center gap-2.5 transition-transform">
							<div
								class="relative flex h-8 w-8 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-sm shadow-primary/25 transition-transform duration-300 group-hover:scale-105"
							>
								<svg viewBox="0 0 24 24" class="h-4 w-4 fill-current" aria-hidden="true">
									<circle cx="12" cy="12" r="3" />
									<path
										d="M12 2v3m0 14v3M2 12h3m14 0h3m-3.5-6.5-2.1 2.1m-8.8 8.8-2.1 2.1m0-13 2.1 2.1m8.8 8.8 2.1 2.1"
										stroke="currentColor"
										stroke-width="2"
										stroke-linecap="round"
										fill="none"
									/>
								</svg>
							</div>
							<span class="hidden text-xl font-bold tracking-tight text-foreground sm:block">
								Aura<span class="text-primary font-medium">hub</span>
							</span>
						</a>
					</div>

					<!-- Search (Desktop) -->
					<div class="mx-8 hidden max-w-xl flex-1 md:flex">
						<SearchAutocomplete />
					</div>

					<!-- Actions -->
					<div class="flex items-center gap-2 sm:gap-3">
						<!-- Mobile Search Toggle -->
						<button
							onclick={() => (isMobileSearchOpen = true)}
							class="hover:bg-secondary/80 rounded-full p-2 text-muted-foreground hover:text-foreground md:hidden"
							aria-label="Open search"
						>
							<Search class="h-5 w-5" />
						</button>

						<a
							href="/upload"
							class="hidden sm:flex items-center gap-2 rounded-full border border-border/80 bg-secondary/60 hover:bg-secondary px-3.5 py-1.5 text-xs font-semibold text-foreground transition-all hover:border-primary/40 shadow-xs"
						>
							<PlusCircle class="h-3.5 w-3.5 text-primary" />
							<span>Upload</span>
						</a>

						<button
							onclick={toggleTheme}
							class="hover:bg-secondary/80 rounded-full p-2 text-muted-foreground hover:text-foreground transition-colors"
							aria-label="Toggle theme"
						>
							{#if isDark}
								<Sun class="h-4 w-4" />
							{:else}
								<Moon class="h-4 w-4" />
							{/if}
						</button>

						<!-- Content Mode Toggle (General Safe vs 18+ Adult) - Only visible when enabled on /profile -->
						{#if userState.isLoaded && userState.user?.showAdultContent}
							<button
								onclick={handleToggleContentMode}
								class="flex items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs font-semibold transition-all duration-200 select-none {contentModeState.isAdult
									? 'border-rose-500/50 bg-rose-500/15 text-rose-400 shadow-xs shadow-rose-500/20 hover:bg-rose-500/25'
									: 'border-border/80 bg-secondary/50 text-muted-foreground hover:bg-secondary hover:text-foreground'}"
								title={contentModeState.isAdult
									? 'Adult Mode Active (18+) - Click to switch to General Content'
									: 'General Safe Mode - Click to switch to 18+ Content'}
								aria-label="Toggle content mode"
							>
								{#if contentModeState.isAdult}
									<span class="inline-block h-2 w-2 rounded-full bg-rose-500 animate-pulse"></span>
									<span>18+</span>
								{:else}
									<ShieldCheck class="h-3.5 w-3.5 text-emerald-400" />
									<span class="hidden sm:inline">Safe</span>
								{/if}
							</button>
						{/if}

						{#if userState.isLoaded}
							{#if userState.user}
								<NotificationBell />
								<div class="ml-1 flex items-center gap-3">
									{#if userState.user.isAdmin}
										<a
											href="/admin"
											class="hidden items-center gap-1.5 rounded-full border border-amber-500/40 bg-amber-500/15 px-2.5 py-1 text-xs font-bold text-amber-400 transition-colors hover:bg-amber-500/25 sm:flex"
											title="Admin Console"
										>
											<ShieldCheck class="h-3.5 w-3.5" />
											<span>Admin</span>
										</a>
									{/if}
									<a
										href="/dashboard"
										class="hidden rounded-full border border-primary/20 bg-primary/10 px-2.5 py-1 text-xs font-semibold text-primary transition-colors hover:bg-primary/20 sm:block"
									>
										Studio
									</a>
									<a
										href="/profile/{userState.user.username}"
										class="flex items-center gap-2 rounded-full py-0.5 pr-2 pl-0.5 transition-opacity hover:opacity-85"
									>
										<img
											src={userState.user.avatar ||
												`https://api.dicebear.com/7.x/identicon/svg?seed=${userState.user.username}`}
											alt={userState.user.username}
											class="h-8 w-8 rounded-full border border-border object-cover"
										/>
										<span class="hidden text-xs font-semibold text-foreground sm:block"
											>@{userState.user.username}</span
										>
									</a>
									<button
										onclick={handleLogout}
										class="hover:bg-secondary/80 rounded-full p-2 text-rose-400 transition-colors"
										aria-label="Log out"
									>
										<LogOut class="h-4 w-4" />
									</button>
								</div>
							{:else}
								<a
									href="/login"
									class="hidden sm:inline-flex h-8 items-center justify-center rounded-full border border-border/80 px-4 text-xs font-semibold text-foreground transition-colors hover:bg-secondary/70 hover:border-primary/40"
								>
									Sign in
								</a>
								<a
									href="/signup"
									class="inline-flex h-8 items-center justify-center rounded-full bg-primary text-primary-foreground font-semibold px-4 text-xs shadow-xs transition-all hover:bg-primary/90 hover:shadow-primary/20"
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
				class={`absolute inset-y-0 left-0 z-40 w-64 shrink-0 transform overflow-y-auto border-r border-border/60 bg-background/95 backdrop-blur-xl transition-transform duration-200 ease-in-out lg:static ${isSidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'}`}
			>
				<nav class="space-y-1.5 p-4 text-sm font-medium">
					<a
						href="/"
						onclick={() => (isSidebarOpen = false)}
						class={`flex items-center gap-3 rounded-xl px-3.5 py-2.5 transition-all ${
							isActive('/')
								? 'bg-primary/15 text-primary border border-primary/25 font-semibold shadow-xs'
								: 'text-muted-foreground hover:bg-secondary/60 hover:text-foreground'
						}`}
					>
						<Home class="h-4 w-4 {isActive('/') ? 'text-primary' : ''}" /> Home
					</a>
					<a
						href="/shorts"
						onclick={() => (isSidebarOpen = false)}
						class={`flex items-center gap-3 rounded-xl px-3.5 py-2.5 transition-all ${
							isActive('/shorts')
								? 'bg-primary/15 text-primary border border-primary/25 font-semibold shadow-xs'
								: 'text-muted-foreground hover:bg-secondary/60 hover:text-foreground'
						}`}
					>
						<Smartphone class="h-4 w-4 {isActive('/shorts') ? 'text-primary' : ''}" /> Shorts
					</a>
					<a
						href="/subscriptions"
						onclick={() => (isSidebarOpen = false)}
						class={`flex items-center gap-3 rounded-xl px-3.5 py-2.5 transition-all ${
							isActive('/subscriptions')
								? 'bg-primary/15 text-primary border border-primary/25 font-semibold shadow-xs'
								: 'text-muted-foreground hover:bg-secondary/60 hover:text-foreground'
						}`}
					>
						<PlaySquare class="h-4 w-4 {isActive('/subscriptions') ? 'text-primary' : ''}" /> Subscriptions
					</a>

					<div class="my-3 border-t border-border/50"></div>
					<h3 class="text-muted-foreground/80 mb-1 px-3 text-[11px] font-bold uppercase tracking-wider">You</h3>
					<a
						href="/notifications"
						onclick={() => (isSidebarOpen = false)}
						class={`flex items-center gap-3 rounded-xl px-3.5 py-2.5 transition-all ${
							isActive('/notifications')
								? 'bg-primary/15 text-primary border border-primary/25 font-semibold shadow-xs'
								: 'text-muted-foreground hover:bg-secondary/60 hover:text-foreground'
						}`}
					>
						<Bell class="h-4 w-4 {isActive('/notifications') ? 'text-primary' : ''}" /> Notifications
					</a>
					<a
						href="/history"
						onclick={() => (isSidebarOpen = false)}
						class={`flex items-center gap-3 rounded-xl px-3.5 py-2.5 transition-all ${
							isActive('/history')
								? 'bg-primary/15 text-primary border border-primary/25 font-semibold shadow-xs'
								: 'text-muted-foreground hover:bg-secondary/60 hover:text-foreground'
						}`}
					>
						<Clock class="h-4 w-4 {isActive('/history') ? 'text-primary' : ''}" /> History
					</a>
					<a
						href="/watch-later"
						onclick={() => (isSidebarOpen = false)}
						class={`flex items-center gap-3 rounded-xl px-3.5 py-2.5 transition-all ${
							isActive('/watch-later')
								? 'bg-primary/15 text-primary border border-primary/25 font-semibold shadow-xs'
								: 'text-muted-foreground hover:bg-secondary/60 hover:text-foreground'
						}`}
					>
						<Clock class="h-4 w-4 {isActive('/watch-later') ? 'text-primary' : ''}" /> Watch Later
					</a>
					<a
						href="/my-playlists"
						onclick={() => (isSidebarOpen = false)}
						class={`flex items-center gap-3 rounded-xl px-3.5 py-2.5 transition-all ${
							isActive('/my-playlists') || isActive('/playlist')
								? 'bg-primary/15 text-primary border border-primary/25 font-semibold shadow-xs'
								: 'text-muted-foreground hover:bg-secondary/60 hover:text-foreground'
						}`}
					>
						<ListVideo class="h-4 w-4 {isActive('/my-playlists') || isActive('/playlist') ? 'text-primary' : ''}" /> Playlists
					</a>

					<div class="my-3 border-t border-border/50"></div>
					<h3 class="text-muted-foreground/80 mb-1 px-3 text-[11px] font-bold uppercase tracking-wider">Creator Studio</h3>
					<a
						href="/dashboard"
						onclick={() => (isSidebarOpen = false)}
						class={`flex items-center gap-3 rounded-xl px-3.5 py-2.5 transition-all ${
							isActive('/dashboard')
								? 'bg-primary/15 text-primary border border-primary/25 font-semibold shadow-xs'
								: 'text-muted-foreground hover:bg-secondary/60 hover:text-foreground'
						}`}
					>
						<Video class="h-4 w-4 {isActive('/dashboard') ? 'text-primary' : ''}" /> Content
					</a>
					<a
						href="/analytics"
						onclick={() => (isSidebarOpen = false)}
						class={`flex items-center gap-3 rounded-xl px-3.5 py-2.5 transition-all ${
							isActive('/analytics')
								? 'bg-primary/15 text-primary border border-primary/25 font-semibold shadow-xs'
								: 'text-muted-foreground hover:bg-secondary/60 hover:text-foreground'
						}`}
					>
						<BarChart class="h-4 w-4 {isActive('/analytics') ? 'text-primary' : ''}" /> Analytics
					</a>

					{#if userState.isLoaded && userState.user?.isAdmin}
						<div class="my-3 border-t border-border/50"></div>
						<h3 class="text-amber-500/90 mb-1 flex items-center gap-1.5 px-3 text-[11px] font-bold uppercase tracking-wider">
							<ShieldCheck class="h-3.5 w-3.5" /> Administration
						</h3>
						<a
							href="/admin"
							onclick={() => (isSidebarOpen = false)}
							class={`flex items-center gap-3 rounded-xl px-3.5 py-2.5 transition-all ${
								isActive('/admin')
									? 'bg-amber-500/20 text-amber-400 border border-amber-500/30 font-semibold'
									: 'text-amber-500/80 hover:bg-amber-500/10 hover:text-amber-400'
							}`}
						>
							<Shield class="h-4 w-4" /> Admin Console
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
			class="fixed right-0 bottom-0 left-0 z-50 flex h-16 items-center justify-around border-t border-border/60 bg-background/90 backdrop-blur-xl px-2 text-xs font-medium pb-safe md:hidden"
		>
			<a
				href="/"
				class={`flex flex-col items-center gap-1 p-2 transition-colors ${
					isActive('/') ? 'text-primary font-semibold' : 'text-muted-foreground hover:text-foreground'
				}`}
			>
				<Home class="h-5 w-5" />
				<span>Home</span>
			</a>
			<a
				href="/shorts"
				class={`flex flex-col items-center gap-1 p-2 transition-colors ${
					isActive('/shorts') ? 'text-primary font-semibold' : 'text-muted-foreground hover:text-foreground'
				}`}
			>
				<Smartphone class="h-5 w-5" />
				<span>Shorts</span>
			</a>
			<a
				href="/upload"
				class="hover:text-foreground flex flex-col items-center gap-1 p-2 transition-colors"
			>
				<div class="text-primary-foreground -mt-2 rounded-full bg-primary p-1 shadow-md shadow-primary/25">
					<PlusCircle class="h-5 w-5" />
				</div>
				<span class="text-foreground">Upload</span>
			</a>
			<a
				href="/subscriptions"
				class={`flex flex-col items-center gap-1 p-2 transition-colors ${
					isActive('/subscriptions') ? 'text-primary font-semibold' : 'text-muted-foreground hover:text-foreground'
				}`}
			>
				<PlaySquare class="h-5 w-5" />
				<span>Subs</span>
			</a>
			{#if userState.isLoaded && userState.user}
				<a
					href="/profile/{userState.user.username}"
					class={`flex flex-col items-center gap-1 p-2 transition-colors ${
						isActive(`/profile/${userState.user.username}`) ? 'text-primary font-semibold' : 'text-muted-foreground hover:text-foreground'
					}`}
				>
					<img
						src={userState.user.avatar ||
							`https://api.dicebear.com/7.x/identicon/svg?seed=${userState.user.username}`}
						alt={userState.user.username}
						class="h-5 w-5 rounded-full border border-border object-cover"
					/>
					<span>You</span>
				</a>
			{:else}
				<a
					href="/login"
					class={`flex flex-col items-center gap-1 p-2 transition-colors ${
						isActive('/login') ? 'text-primary font-semibold' : 'text-muted-foreground hover:text-foreground'
					}`}
				>
					<User class="h-5 w-5" />
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
