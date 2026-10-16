<script lang="ts">
	import { userState } from '#lib/user.svelte';
	import { fetchApi } from '#lib/api';
	import { createQuery, useQueryClient } from '@tanstack/svelte-query';
	import {
		Shield,
		ShieldCheck,
		ShieldAlert,
		Users,
		Film,
		Eye,
		RefreshCw,
		Trash2,
		Search,
		Filter,
		Server,
		Database,
		Activity,
		CheckCircle2,
		XCircle,
		AlertTriangle,
		Crown,
		ExternalLink,
		Flame,
		Radio,
		Layers,
		Loader2,
		Play,
		Check,
		ArrowRight,
		Zap,
		Power,
		Lock,
		Upload,
		MessageSquare,
		AlertOctagon
	} from 'lucide-svelte';
	import { Button } from '#lib/components/ui/button';
	import { Input } from '#lib/components/ui/input';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { goto } from '$app/navigation';

	const queryClient = useQueryClient();

	let activeTab = $state<'overview' | 'videos' | 'users' | 'operations' | 'killswitches'>('overview');

	// Notification banner state
	let toastMessage = $state<{ type: 'success' | 'error'; text: string } | null>(null);
	function showToast(text: string, type: 'success' | 'error' = 'success') {
		toastMessage = { type, text };
		setTimeout(() => {
			if (toastMessage?.text === text) toastMessage = null;
		}, 4500);
	}

	// === Tab 1: Stats Query ===
	const statsQuery = createQuery(
		() => ({
			queryKey: ['admin', 'stats'],
			queryFn: async () => {
				return await fetchApi('/admin/stats');
			},
			enabled: !!userState.user?.isAdmin
		}),
		() => queryClient
	);

	function getSafePct(s: any): number {
		return s?.videos?.total > 0 ? Math.round((s.videos.safe / s.videos.total) * 100) : 100;
	}
	function getFreshCount(s: any): number {
		return Math.max(0, (s?.videos?.total || 0) - (s?.streamtape?.needingRefresh || 0));
	}
	function getFreshPct(s: any): number {
		const total = s?.videos?.total || 0;
		return total > 0 ? Math.round((getFreshCount(s) / total) * 100) : 100;
	}

	// === Tab 2: Videos State & Query ===
	let videoSearch = $state('');
	let videoAdultFilter = $state<'all' | 'true' | 'false'>('all');
	let videoStatusFilter = $state<string>('all');
	let videoVisibilityFilter = $state<string>('all');
	let videoSort = $state<'newest' | 'views' | 'needs_refresh'>('newest');
	let videoPage = $state(1);

	let videoDebounceTimer: ReturnType<typeof setTimeout> | null = null;
	let activeVideoSearch = $state('');

	function handleVideoSearchInput(e: Event) {
		const val = (e.target as HTMLInputElement).value;
		videoSearch = val;
		if (videoDebounceTimer) clearTimeout(videoDebounceTimer);
		videoDebounceTimer = setTimeout(() => {
			activeVideoSearch = val.trim();
			videoPage = 1;
		}, 250);
	}

	const videosQuery = createQuery(
		() => ({
			queryKey: [
				'admin',
				'videos',
				videoPage,
				activeVideoSearch,
				videoAdultFilter,
				videoStatusFilter,
				videoVisibilityFilter,
				videoSort
			],
			queryFn: async () => {
				const params = new URLSearchParams({
					page: String(videoPage),
					limit: '15',
					q: activeVideoSearch,
					sort: videoSort
				});
				if (videoAdultFilter !== 'all') params.set('adult', videoAdultFilter);
				if (videoStatusFilter !== 'all') params.set('status', videoStatusFilter);
				if (videoVisibilityFilter !== 'all') params.set('visibility', videoVisibilityFilter);

				return await fetchApi(`/admin/videos?${params.toString()}`);
			},
			enabled: !!userState.user?.isAdmin && activeTab === 'videos'
		}),
		() => queryClient
	);

	// Video moderation action: Toggle isAdult
	let updatingVideoId = $state<string | null>(null);
	async function toggleVideoAdult(videoId: string, currentAdult: boolean) {
		updatingVideoId = videoId;
		try {
			await fetchApi(`/admin/videos/${videoId}`, {
				method: 'PUT',
				body: JSON.stringify({ isAdult: !currentAdult })
			});
			showToast(
				`Video updated to ${!currentAdult ? '18+ Adult' : 'Safe General'} classification.`
			);
			queryClient.invalidateQueries({ queryKey: ['admin', 'videos'] });
			queryClient.invalidateQueries({ queryKey: ['admin', 'stats'] });
		} catch (err: any) {
			showToast(err.message || 'Failed to update video classification', 'error');
		} finally {
			updatingVideoId = null;
		}
	}

	// Video moderation action: Change Visibility
	async function changeVideoVisibility(videoId: string, newVisibility: string) {
		updatingVideoId = videoId;
		try {
			await fetchApi(`/admin/videos/${videoId}`, {
				method: 'PUT',
				body: JSON.stringify({ visibility: newVisibility })
			});
			showToast(`Video visibility changed to ${newVisibility}.`);
			queryClient.invalidateQueries({ queryKey: ['admin', 'videos'] });
		} catch (err: any) {
			showToast(err.message || 'Failed to update visibility', 'error');
		} finally {
			updatingVideoId = null;
		}
	}

	// Video moderation action: Delete Video
	let deletingVideoId = $state<string | null>(null);
	async function deleteVideo(videoId: string, title: string) {
		if (!confirm(`Are you sure you want to permanently delete "${title}" platform-wide?`)) {
			return;
		}
		deletingVideoId = videoId;
		try {
			await fetchApi(`/admin/videos/${videoId}`, {
				method: 'DELETE'
			});
			showToast(`Video "${title}" deleted successfully.`);
			queryClient.invalidateQueries({ queryKey: ['admin', 'videos'] });
			queryClient.invalidateQueries({ queryKey: ['admin', 'stats'] });
		} catch (err: any) {
			showToast(err.message || 'Failed to delete video', 'error');
		} finally {
			deletingVideoId = null;
		}
	}

	// === Tab 3: Users State & Query ===
	let userSearch = $state('');
	let activeUserSearch = $state('');
	let userPage = $state(1);
	let userDebounceTimer: ReturnType<typeof setTimeout> | null = null;

	function handleUserSearchInput(e: Event) {
		const val = (e.target as HTMLInputElement).value;
		userSearch = val;
		if (userDebounceTimer) clearTimeout(userDebounceTimer);
		userDebounceTimer = setTimeout(() => {
			activeUserSearch = val.trim();
			userPage = 1;
		}, 250);
	}

	const usersQuery = createQuery(
		() => ({
			queryKey: ['admin', 'users', userPage, activeUserSearch],
			queryFn: async () => {
				const params = new URLSearchParams({
					page: String(userPage),
					limit: '15',
					q: activeUserSearch
				});
				return await fetchApi(`/admin/users?${params.toString()}`);
			},
			enabled: !!userState.user?.isAdmin && activeTab === 'users'
		}),
		() => queryClient
	);

	// User role action: Toggle Admin
	let updatingUserId = $state<string | null>(null);
	async function toggleUserAdmin(userId: string, currentAdmin: boolean, username: string) {
		if (
			!confirm(
				`Are you sure you want to ${currentAdmin ? 'REVOKE admin rights from' : 'GRANT admin privileges to'} ${username}?`
			)
		) {
			return;
		}
		updatingUserId = userId;
		try {
			await fetchApi(`/admin/users/${userId}/role`, {
				method: 'PUT',
				body: JSON.stringify({ isAdmin: !currentAdmin })
			});
			showToast(`Admin privileges ${!currentAdmin ? 'granted to' : 'revoked from'} ${username}.`);
			queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
		} catch (err: any) {
			showToast(err.message || 'Failed to update user role', 'error');
		} finally {
			updatingUserId = null;
		}
	}

	// User deletion action
	let deletingUserId = $state<string | null>(null);
	async function deleteUser(userId: string, username: string) {
		if (
			!confirm(
				`CRITICAL: Are you sure you want to delete user account "${username}"? All their uploads and playlists will be permanently deleted.`
			)
		) {
			return;
		}
		deletingUserId = userId;
		try {
			await fetchApi(`/admin/users/${userId}`, {
				method: 'DELETE'
			});
			showToast(`User account "${username}" deleted.`);
			queryClient.invalidateQueries({ queryKey: ['admin', 'users'] });
			queryClient.invalidateQueries({ queryKey: ['admin', 'stats'] });
		} catch (err: any) {
			showToast(err.message || 'Failed to delete user', 'error');
		} finally {
			deletingUserId = null;
		}
	}

	// === Tab 4: System Operations ===
	let isRunningRefreshCycle = $state(false);
	let refreshCycleLogs = $state<string[]>([]);
	let refreshDaysThreshold = $state(28);
	let refreshBatchLimit = $state(25);

	async function runRefreshCycle() {
		isRunningRefreshCycle = true;
		refreshCycleLogs = [
			`[${new Date().toLocaleTimeString()}] Triggering manual remote upload renewal cycle (Threshold: ${refreshDaysThreshold} days, Batch: ${refreshBatchLimit})...`
		];
		try {
			const res = await fetchApi(
				`/admin/actions/refresh-cycle?days=${refreshDaysThreshold}&limit=${refreshBatchLimit}`,
				{
					method: 'POST'
				}
			);
			refreshCycleLogs = [
				...refreshCycleLogs,
				`[${new Date().toLocaleTimeString()}] ${res.message || 'Success'}`,
				`[${new Date().toLocaleTimeString()}] Refreshed ${res.refreshed} video(s) upstream on Streamtape storage.`
			];
			showToast(`Renewal cycle finished: ${res.refreshed} video(s) renewed.`);
			queryClient.invalidateQueries({ queryKey: ['admin', 'stats'] });
		} catch (err: any) {
			refreshCycleLogs = [
				...refreshCycleLogs,
				`[${new Date().toLocaleTimeString()}] ERROR: ${err.message || 'Failed to execute refresh cycle'}`
			];
			showToast(err.message || 'Failed to execute renewal cycle', 'error');
		} finally {
			isRunningRefreshCycle = false;
		}
	}

	let isFlushingCache = $state(false);
	async function flushApiCache() {
		isFlushingCache = true;
		try {
			await fetchApi('/admin/actions/flush-cache', { method: 'POST' });
			showToast('Valkey/Redis cache flushed successfully across all API keys.');
		} catch (err: any) {
			showToast(err.message || 'Failed to flush cache', 'error');
		} finally {
			isFlushingCache = false;
		}
	}

	function formatNumber(num: number | undefined): string {
		if (num === undefined || num === null) return '0';
		return new Intl.NumberFormat().format(num);
	}

	function formatDate(dateStr: string | undefined): string {
		if (!dateStr) return 'N/A';
		const d = new Date(dateStr);
		return d.toLocaleDateString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}

	function formatDateTime(dateStr: string | undefined): string {
		if (!dateStr) return 'Default (System Start)';
		const d = new Date(dateStr);
		return d.toLocaleString(undefined, {
			month: 'short',
			day: 'numeric',
			hour: '2-digit',
			minute: '2-digit',
			second: '2-digit'
		});
	}

	// === Tab 5: Kill Switches & Superpowers ===
	interface KillSwitch {
		key: string;
		name?: string;
		description?: string;
		is_active?: boolean;
		enabled?: boolean;
		updated_by?: string;
		updatedBy?: string;
		updated_at?: string;
		updatedAt?: string;
	}

	const killSwitchesQuery = createQuery<Record<string, KillSwitch>>(
		() => ({
			queryKey: ['admin', 'kill-switches'],
			queryFn: async () => {
				const res = await fetchApi('/admin/kill-switches');
				return res?.switches || res || {};
			},
			enabled: !!userState.user?.isAdmin,
			refetchInterval: 5000
		}),
		() => queryClient
	);

	let togglingKey = $state<string | null>(null);

	async function toggleKillSwitch(key: string, currentState: boolean) {
		togglingKey = key;
		const newState = !currentState;
		try {
			await fetchApi('/admin/kill-switches', {
				method: 'POST',
				body: JSON.stringify({
					key,
					enabled: newState,
					is_active: newState
				})
			});
			showToast(`Superpower kill switch "${key}" is now ${newState ? 'ARMED' : 'DISARMED'}.`);
			await queryClient.invalidateQueries({ queryKey: ['admin', 'kill-switches'] });
			await queryClient.refetchQueries({ queryKey: ['admin', 'kill-switches'] });
		} catch (err: any) {
			showToast(err.message || 'Failed to update kill switch', 'error');
		} finally {
			togglingKey = null;
		}
	}

	let isBulkOperating = $state(false);
	async function bulkSetKillSwitches(activateAll: boolean) {
		const actionPrompt = activateAll
			? 'ARE YOU SURE? This will immediately trigger TOTAL PLATFORM LOCKDOWN, freezing all uploads, streaming, registrations, and comments.'
			: 'Are you sure you want to DISARM all kill switches and restore standard platform operations?';
		if (!confirm(actionPrompt)) {
			return;
		}
		isBulkOperating = true;
		const keys = ['maintenance_mode', 'freeze_uploads', 'freeze_streaming', 'freeze_registration', 'freeze_comments'];
		try {
			for (const key of keys) {
				await fetchApi('/admin/kill-switches', {
					method: 'POST',
					body: JSON.stringify({ key, enabled: activateAll, is_active: activateAll })
				});
			}
			showToast(`All kill switches have been ${activateAll ? 'ARMED' : 'DISARMED'} successfully.`);
			await queryClient.invalidateQueries({ queryKey: ['admin', 'kill-switches'] });
			await queryClient.refetchQueries({ queryKey: ['admin', 'kill-switches'] });
		} catch (err: any) {
			showToast(err.message || 'Failed bulk kill switch update', 'error');
		} finally {
			isBulkOperating = false;
		}
	}

	function isSwitchActive(sw: KillSwitch | undefined): boolean {
		if (!sw) return false;
		return Boolean(sw.is_active || sw.enabled);
	}

	function countActiveKillSwitches(switches: Record<string, KillSwitch> | undefined): number {
		if (!switches) return 0;
		return Object.values(switches).filter((s) => isSwitchActive(s)).length;
	}

	let maintenance = $derived(killSwitchesQuery.data?.['maintenance_mode']);
	let uploads = $derived(killSwitchesQuery.data?.['freeze_uploads']);
	let streaming = $derived(killSwitchesQuery.data?.['freeze_streaming']);
	let reg = $derived(killSwitchesQuery.data?.['freeze_registration']);
	let comments = $derived(killSwitchesQuery.data?.['freeze_comments']);

	let isMaintenanceActive = $derived(isSwitchActive(maintenance));
	let isUploadsActive = $derived(isSwitchActive(uploads));
	let isStreamingActive = $derived(isSwitchActive(streaming));
	let isRegistrationActive = $derived(isSwitchActive(reg));
	let isCommentsActive = $derived(isSwitchActive(comments));
</script>

<svelte:head>
	<title>Admin Dashboard - Aurahub</title>
</svelte:head>

{#if !userState.isLoaded}
	<div class="flex h-96 items-center justify-center">
		<Loader2 class="h-8 w-8 animate-spin text-primary" />
	</div>
{:else if !userState.user?.isAdmin}
	<!-- Access Denied -->
	<div class="mx-auto max-w-xl py-16 text-center">
		<div class="rounded-3xl border border-destructive/30 bg-destructive/10 p-8 shadow-xl">
			<div class="mx-auto mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-destructive/20 text-destructive">
				<ShieldAlert class="h-8 w-8" />
			</div>
			<h1 class="text-2xl font-bold text-foreground">Access Restricted</h1>
			<p class="mt-2 text-sm text-muted-foreground leading-relaxed">
				You must have administrator privileges to access the Aurahub Admin Console. If you believe this is an error, contact the system administrator.
			</p>
			<div class="mt-6 flex justify-center gap-3">
				<Button onclick={() => goto('/')} variant="outline" class="rounded-full">
					Return to Feed
				</Button>
				<Button onclick={() => goto('/profile')} class="rounded-full">
					My Profile
				</Button>
			</div>
		</div>
	</div>
{:else}
	<!-- Admin Panel Main Content -->
	<div class="space-y-6">
		<!-- Header Banner -->
		<div class="relative overflow-hidden rounded-3xl border border-amber-500/20 bg-gradient-to-r from-amber-500/10 via-background to-rose-500/10 p-6 shadow-xl md:p-8">
			<div class="relative z-10 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
				<div class="space-y-1.5">
					<div class="inline-flex items-center gap-2 rounded-full border border-amber-500/30 bg-amber-500/15 px-3 py-1 text-xs font-bold text-amber-500">
						<Crown class="h-3.5 w-3.5" />
						<span>Aurahub Operations Control</span>
					</div>
					<h1 class="text-2xl font-black tracking-tight text-foreground md:text-3xl">
						Admin Dashboard & Moderation
					</h1>
					<p class="text-xs text-muted-foreground md:text-sm">
						Platform-wide content moderation, video renewal lifecycle, user management, and infrastructure diagnostics.
					</p>
				</div>

				<!-- Live Status Pills & Quick Actions -->
				<div class="flex flex-wrap items-center gap-2.5">
					<div class="flex items-center gap-1.5 rounded-full border border-emerald-500/30 bg-emerald-500/10 px-3 py-1.5 text-xs font-semibold text-emerald-400">
						<span class="h-2 w-2 rounded-full bg-emerald-400 animate-pulse"></span>
						<span>System Healthy</span>
					</div>
					<Button
						variant="outline"
						size="sm"
						onclick={() => queryClient.invalidateQueries()}
						class="rounded-full gap-1.5 text-xs font-medium"
					>
						<RefreshCw class="h-3.5 w-3.5" />
						<span>Sync Data</span>
					</Button>
				</div>
			</div>
		</div>

		<!-- Toast Notification -->
		{#if toastMessage}
			<div
				class="animate-in fade-in slide-in-from-top-2 flex items-center justify-between rounded-2xl border px-4 py-3 text-sm font-semibold shadow-lg duration-200 {toastMessage.type === 'success' ? 'border-emerald-500/40 bg-emerald-500/15 text-emerald-400' : 'border-destructive/40 bg-destructive/15 text-destructive'}"
			>
				<div class="flex items-center gap-2.5">
					{#if toastMessage.type === 'success'}
						<CheckCircle2 class="h-4 w-4 shrink-0 text-emerald-400" />
					{:else}
						<AlertTriangle class="h-4 w-4 shrink-0 text-destructive" />
					{/if}
					<span>{toastMessage.text}</span>
				</div>
				<button
					type="button"
					onclick={() => (toastMessage = null)}
					class="text-xs opacity-70 hover:opacity-100"
				>
					✕
				</button>
			</div>
		{/if}

		<!-- Navigation Tabs -->
		<div class="hide-scrollbar flex items-center gap-2 overflow-x-auto border-b pb-1">
			<button
				type="button"
				onclick={() => (activeTab = 'overview')}
				class="flex items-center gap-2 rounded-xl px-4 py-2.5 text-sm font-semibold transition-all {activeTab === 'overview' ? 'bg-primary text-primary-foreground shadow-md' : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
			>
				<Activity class="h-4 w-4" />
				<span>Overview & KPIs</span>
			</button>
			<button
				type="button"
				onclick={() => (activeTab = 'videos')}
				class="flex items-center gap-2 rounded-xl px-4 py-2.5 text-sm font-semibold transition-all {activeTab === 'videos' ? 'bg-primary text-primary-foreground shadow-md' : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
			>
				<Film class="h-4 w-4" />
				<span>Video Moderation</span>
				{#if statsQuery.data?.videos?.total}
					<span class="rounded-full bg-background/20 px-2 py-0.5 text-xs font-bold">
						{statsQuery.data.videos.total}
					</span>
				{/if}
			</button>
			<button
				type="button"
				onclick={() => (activeTab = 'users')}
				class="flex items-center gap-2 rounded-xl px-4 py-2.5 text-sm font-semibold transition-all {activeTab === 'users' ? 'bg-primary text-primary-foreground shadow-md' : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
			>
				<Users class="h-4 w-4" />
				<span>User Directory</span>
				{#if statsQuery.data?.users?.total}
					<span class="rounded-full bg-background/20 px-2 py-0.5 text-xs font-bold">
						{statsQuery.data.users.total}
					</span>
				{/if}
			</button>
			<button
				type="button"
				onclick={() => (activeTab = 'operations')}
				class="flex items-center gap-2 rounded-xl px-4 py-2.5 text-sm font-semibold transition-all {activeTab === 'operations' ? 'bg-primary text-primary-foreground shadow-md' : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
			>
				<Server class="h-4 w-4" />
				<span>Operations & Lifecycle</span>
				{#if statsQuery.data?.streamtape?.needingRefresh > 0}
					<span class="rounded-full bg-amber-500/20 px-2 py-0.5 text-xs font-bold text-amber-400">
						{statsQuery.data.streamtape.needingRefresh} due
					</span>
				{/if}
			</button>
			<button
				type="button"
				onclick={() => (activeTab = 'killswitches')}
				class="flex items-center gap-2 rounded-xl px-4 py-2.5 text-sm font-semibold transition-all {activeTab === 'killswitches' ? 'bg-primary text-primary-foreground shadow-md' : 'text-muted-foreground hover:bg-muted hover:text-foreground'}"
			>
				<Zap class="h-4 w-4 {countActiveKillSwitches(killSwitchesQuery.data) > 0 ? 'text-amber-400 fill-amber-400 animate-pulse' : ''}" />
				<span>Kill Switches & Superpowers</span>
				{#if countActiveKillSwitches(killSwitchesQuery.data) > 0}
					<span class="rounded-full bg-destructive/30 border border-destructive/50 px-2 py-0.5 text-xs font-bold text-destructive">
						{countActiveKillSwitches(killSwitchesQuery.data)} ARMED
					</span>
				{:else}
					<span class="rounded-full bg-emerald-500/20 px-2 py-0.5 text-xs font-bold text-emerald-400">
						NORMAL
					</span>
				{/if}
			</button>
		</div>

		<!-- ================= TAB 1: OVERVIEW & KPIS ================= -->
		{#if activeTab === 'overview'}
			<div class="space-y-6">
				<!-- KPI Metric Cards Grid -->
				{#if statsQuery.isPending}
					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
						{#each Array(4) as _}
							<Skeleton class="h-32 rounded-3xl" />
						{/each}
					</div>
				{:else if statsQuery.data}
					{@const s = statsQuery.data}
					<div class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
						<!-- Card 1: Users -->
						<div class="group relative overflow-hidden rounded-3xl border border-border/80 bg-card p-6 shadow-sm transition-all hover:border-primary/40 hover:shadow-lg">
							<div class="flex items-center justify-between">
								<span class="text-xs font-bold tracking-wider uppercase text-muted-foreground">Registered Users</span>
								<div class="rounded-2xl bg-blue-500/10 p-2.5 text-blue-500">
									<Users class="h-5 w-5" />
								</div>
							</div>
							<p class="mt-3 text-3xl font-black tracking-tight text-foreground">{formatNumber(s.users.total)}</p>
							<div class="mt-2 flex items-center justify-between text-xs text-muted-foreground">
								<span>18+ Enabled: <strong class="text-foreground">{s.users.adultEnabled}</strong></span>
								<span class="text-emerald-500">+{s.users.recentSignups} this wk</span>
							</div>
						</div>

						<!-- Card 2: Videos -->
						<div class="group relative overflow-hidden rounded-3xl border border-border/80 bg-card p-6 shadow-sm transition-all hover:border-primary/40 hover:shadow-lg">
							<div class="flex items-center justify-between">
								<span class="text-xs font-bold tracking-wider uppercase text-muted-foreground">Total Videos</span>
								<div class="rounded-2xl bg-purple-500/10 p-2.5 text-purple-500">
									<Film class="h-5 w-5" />
								</div>
							</div>
							<p class="mt-3 text-3xl font-black tracking-tight text-foreground">{formatNumber(s.videos.total)}</p>
							<div class="mt-2 flex items-center gap-2 text-xs">
								<span class="rounded-full bg-emerald-500/15 px-2 py-0.5 font-bold text-emerald-500">{s.videos.safe} Safe</span>
								<span class="rounded-full bg-rose-500/15 px-2 py-0.5 font-bold text-rose-500">{s.videos.adult} 18+</span>
							</div>
						</div>

						<!-- Card 3: Views & Plays -->
						<div class="group relative overflow-hidden rounded-3xl border border-border/80 bg-card p-6 shadow-sm transition-all hover:border-primary/40 hover:shadow-lg">
							<div class="flex items-center justify-between">
								<span class="text-xs font-bold tracking-wider uppercase text-muted-foreground">Platform Views</span>
								<div class="rounded-2xl bg-amber-500/10 p-2.5 text-amber-500">
									<Eye class="h-5 w-5" />
								</div>
							</div>
							<p class="mt-3 text-3xl font-black tracking-tight text-foreground">{formatNumber(s.videos.totalViews)}</p>
							<div class="mt-2 flex items-center gap-3 text-xs text-muted-foreground">
								<span>{formatNumber(s.engagement.likes)} Likes</span>
								<span>•</span>
								<span>{formatNumber(s.engagement.comments)} Comments</span>
							</div>
						</div>

						<!-- Card 4: Streamtape Health -->
						<div class="group relative overflow-hidden rounded-3xl border border-border/80 bg-card p-6 shadow-sm transition-all hover:border-primary/40 hover:shadow-lg">
							<div class="flex items-center justify-between">
								<span class="text-xs font-bold tracking-wider uppercase text-muted-foreground">Streamtape Storage</span>
								<div class="rounded-2xl bg-rose-500/10 p-2.5 text-rose-500">
									<Database class="h-5 w-5" />
								</div>
							</div>
							<p class="mt-3 text-3xl font-black tracking-tight text-foreground">
								{formatNumber(s.streamtape.active)} <span class="text-sm font-normal text-muted-foreground">Active</span>
							</p>
							<div class="mt-2 flex items-center justify-between text-xs">
								<span class="text-muted-foreground">Dead: <strong class="text-destructive">{s.streamtape.dead}</strong></span>
								<span class="font-semibold text-amber-400">{s.streamtape.needingRefresh} due refresh</span>
							</div>
						</div>
					</div>

					<!-- Visual Distribution Bars -->
					<div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
						<!-- Content Distribution Card -->
						<div class="rounded-3xl border border-border/80 bg-card p-6 shadow-sm">
							<div class="flex items-center justify-between mb-4">
								<h2 class="text-base font-bold text-foreground">Content Classification Ratio</h2>
								<span class="text-xs text-muted-foreground">Safe vs 18+ Content</span>
							</div>
							<div class="space-y-3">
								<div class="flex h-4 w-full overflow-hidden rounded-full bg-muted">
									<div class="bg-emerald-500 transition-all duration-500" style="width: {getSafePct(s)}%"></div>
									<div class="bg-rose-500 transition-all duration-500" style="width: {100 - getSafePct(s)}%"></div>
								</div>
								<div class="flex items-center justify-between text-xs font-semibold">
									<span class="flex items-center gap-1.5 text-emerald-500">
										<span class="h-2.5 w-2.5 rounded-full bg-emerald-500"></span>
										General Safe: {s.videos.safe} ({getSafePct(s)}%)
									</span>
									<span class="flex items-center gap-1.5 text-rose-500">
										<span class="h-2.5 w-2.5 rounded-full bg-rose-500"></span>
										18+ Mature: {s.videos.adult} ({100 - getSafePct(s)}%)
									</span>
								</div>
							</div>
						</div>

						<!-- Storage Renewal Health Card -->
						<div class="rounded-3xl border border-border/80 bg-card p-6 shadow-sm">
							<div class="flex items-center justify-between mb-4">
								<h2 class="text-base font-bold text-foreground">Streamtape 28-Day Renewal Status</h2>
								<span class="text-xs text-muted-foreground">Anti-Expiration Health</span>
							</div>
							<div class="space-y-3">
								<div class="flex h-4 w-full overflow-hidden rounded-full bg-muted">
									<div class="bg-blue-500 transition-all duration-500" style="width: {getFreshPct(s)}%"></div>
									<div class="bg-amber-500 transition-all duration-500" style="width: {100 - getFreshPct(s)}%"></div>
								</div>
								<div class="flex items-center justify-between text-xs font-semibold">
									<span class="text-blue-400">Fresh / Active: {getFreshCount(s)} ({getFreshPct(s)}%)</span>
									<span class="text-amber-400">Due Renewal (&gt;28d): {s.streamtape.needingRefresh} ({100 - getFreshPct(s)}%)</span>
								</div>
							</div>
						</div>
					</div>

					<!-- Quick Actions Row -->
					<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
						<button
							type="button"
							onclick={() => (activeTab = 'videos')}
							class="flex items-center justify-between rounded-2xl border border-border/80 bg-card p-4 text-left transition-all hover:border-primary hover:bg-muted/40"
						>
							<div class="flex items-center gap-3">
								<div class="rounded-xl bg-purple-500/15 p-2.5 text-purple-400">
									<Film class="h-5 w-5" />
								</div>
								<div>
									<p class="text-sm font-bold text-foreground">Moderate Videos</p>
									<p class="text-xs text-muted-foreground">Recategorize or delete videos</p>
								</div>
							</div>
							<ArrowRight class="h-4 w-4 opacity-50" />
						</button>

						<button
							type="button"
							onclick={() => (activeTab = 'users')}
							class="flex items-center justify-between rounded-2xl border border-border/80 bg-card p-4 text-left transition-all hover:border-primary hover:bg-muted/40"
						>
							<div class="flex items-center gap-3">
								<div class="rounded-xl bg-blue-500/15 p-2.5 text-blue-400">
									<Users class="h-5 w-5" />
								</div>
								<div>
									<p class="text-sm font-bold text-foreground">Manage Users</p>
									<p class="text-xs text-muted-foreground">Permissions and roles</p>
								</div>
							</div>
							<ArrowRight class="h-4 w-4 opacity-50" />
						</button>

						<button
							type="button"
							onclick={() => (activeTab = 'operations')}
							class="flex items-center justify-between rounded-2xl border border-border/80 bg-card p-4 text-left transition-all hover:border-primary hover:bg-muted/40"
						>
							<div class="flex items-center gap-3">
								<div class="rounded-xl bg-amber-500/15 p-2.5 text-amber-400">
									<Server class="h-5 w-5" />
								</div>
								<div>
									<p class="text-sm font-bold text-foreground">28-Day Re-upload Tool</p>
									<p class="text-xs text-muted-foreground">Run remote renewal cycle</p>
								</div>
							</div>
							<ArrowRight class="h-4 w-4 opacity-50" />
						</button>
					</div>
				{/if}
			</div>
		{/if}

		<!-- ================= TAB 2: VIDEO MODERATION ================= -->
		{#if activeTab === 'videos'}
			<div class="space-y-4">
				<!-- Search & Filter Controls -->
				<div class="flex flex-col gap-3 rounded-2xl border border-border/80 bg-card p-4 shadow-sm sm:flex-row sm:items-center sm:justify-between">
					<div class="relative flex-1">
						<Search class="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
						<input
							type="text"
							value={videoSearch}
							oninput={handleVideoSearchInput}
							placeholder="Search videos by title or uploader username..."
							class="w-full rounded-xl border border-border bg-background py-2 pr-4 pl-9 text-sm focus:ring-2 focus:ring-primary/20 focus:outline-none"
						/>
					</div>

					<div class="flex flex-wrap items-center gap-2">
						<!-- Adult Filter -->
						<select
							bind:value={videoAdultFilter}
							onchange={() => (videoPage = 1)}
							class="rounded-xl border border-border bg-background px-3 py-2 text-xs font-semibold focus:outline-none"
						>
							<option value="all">All Content</option>
							<option value="false">🛡️ Safe Only</option>
							<option value="true">🔞 18+ Adult Only</option>
						</select>

						<!-- Streamtape Status Filter -->
						<select
							bind:value={videoStatusFilter}
							onchange={() => (videoPage = 1)}
							class="rounded-xl border border-border bg-background px-3 py-2 text-xs font-semibold focus:outline-none"
						>
							<option value="all">All Stream Status</option>
							<option value="active">🟢 Active</option>
							<option value="pending">🟡 Pending</option>
							<option value="dead">🔴 Dead</option>
						</select>

						<!-- Sort Filter -->
						<select
							bind:value={videoSort}
							onchange={() => (videoPage = 1)}
							class="rounded-xl border border-border bg-background px-3 py-2 text-xs font-semibold focus:outline-none"
						>
							<option value="newest">Newest First</option>
							<option value="views">Most Views</option>
							<option value="needs_refresh">Needs 28d Renewal</option>
						</select>
					</div>
				</div>

				<!-- Video Table -->
				<div class="overflow-hidden rounded-2xl border border-border/80 bg-card shadow-sm">
					{#if videosQuery.isPending}
						<div class="p-8 space-y-3">
							{#each Array(5) as _}
								<Skeleton class="h-16 w-full rounded-xl" />
							{/each}
						</div>
					{:else if videosQuery.data?.videos?.length === 0}
						<div class="py-16 text-center">
							<Film class="mx-auto mb-3 h-10 w-10 text-muted-foreground opacity-40" />
							<p class="text-sm font-semibold text-foreground">No videos match your criteria</p>
							<p class="text-xs text-muted-foreground mt-1">Try modifying your search or filters.</p>
						</div>
					{:else if videosQuery.data}
						<div class="overflow-x-auto">
							<table class="w-full text-left text-xs">
								<thead class="border-b bg-muted/40 font-bold uppercase tracking-wider text-muted-foreground">
									<tr>
										<th class="p-3.5">Video</th>
										<th class="p-3.5">Uploader</th>
										<th class="p-3.5">Views</th>
										<th class="p-3.5">Classification</th>
										<th class="p-3.5">Visibility</th>
										<th class="p-3.5">Streamtape Status</th>
										<th class="p-3.5">Refreshed</th>
										<th class="p-3.5 text-right">Actions</th>
									</tr>
								</thead>
								<tbody class="divide-y divide-border/60">
									{#each videosQuery.data.videos as v}
										<tr class="transition-colors hover:bg-muted/30">
											<!-- Video Info -->
											<td class="p-3.5 min-w-[260px] max-w-[340px]">
												<div class="flex items-center gap-3">
													<div class="relative h-12 w-20 shrink-0 overflow-hidden rounded-lg bg-black">
														{#if v.thumbnailUrl}
															<img src={v.thumbnailUrl} alt="" class="h-full w-full object-cover" />
														{:else}
															<div class="flex h-full w-full items-center justify-center text-muted-foreground">
																<Film class="h-4 w-4 opacity-40" />
															</div>
														{/if}
														{#if v.isShort}
															<span class="absolute bottom-1 right-1 rounded bg-black/80 px-1 text-[9px] font-bold text-white">Short</span>
														{/if}
													</div>
													<div class="min-w-0 flex-1">
														<a
															href="/watch/{v.fileId || v.id}"
															target="_blank"
															class="line-clamp-1 font-semibold text-foreground hover:text-primary transition-colors"
															title={v.title}
														>
															{v.title}
														</a>
														<p class="text-[10px] text-muted-foreground font-mono mt-0.5">
															{v.fileId} • {v.category || 'Other'}
														</p>
													</div>
												</div>
											</td>

											<!-- Uploader -->
											<td class="p-3.5 whitespace-nowrap">
												<div class="flex items-center gap-2">
													<img
														src={v.uploader?.avatar || `https://api.dicebear.com/7.x/identicon/svg?seed=${v.uploader?.username}`}
														alt=""
														class="h-6 w-6 rounded-full object-cover bg-muted"
													/>
													<span class="font-medium text-foreground">{v.uploader?.username || 'Unknown'}</span>
												</div>
											</td>

											<!-- Views -->
											<td class="p-3.5 whitespace-nowrap font-medium text-muted-foreground">
												{formatNumber(v.views)}
											</td>

											<!-- Classification Toggle Button -->
											<td class="p-3.5 whitespace-nowrap">
												<button
													type="button"
													disabled={updatingVideoId === v.id}
													onclick={() => toggleVideoAdult(v.id, v.isAdult)}
													class="inline-flex items-center gap-1.5 rounded-full px-2.5 py-1 text-[11px] font-bold transition-all select-none {v.isAdult
														? 'border border-rose-500/40 bg-rose-500/15 text-rose-500 hover:bg-rose-500/25'
														: 'border border-emerald-500/40 bg-emerald-500/15 text-emerald-500 hover:bg-emerald-500/25'}"
													title="Click to toggle adult status"
												>
													{#if updatingVideoId === v.id}
														<Loader2 class="h-3 w-3 animate-spin" />
													{:else if v.isAdult}
														<span>🔞 18+ Adult</span>
													{:else}
														<span>🛡️ Safe</span>
													{/if}
												</button>
											</td>

											<!-- Visibility Selector -->
											<td class="p-3.5 whitespace-nowrap">
												<select
													value={v.visibility || 'public'}
													onchange={(e) => changeVideoVisibility(v.id, e.currentTarget.value)}
													disabled={updatingVideoId === v.id}
													class="rounded-lg border border-border bg-background px-2 py-1 text-[11px] font-medium focus:outline-none"
												>
													<option value="public">Public</option>
													<option value="unlisted">Unlisted</option>
													<option value="private">Private</option>
												</select>
											</td>

											<!-- Streamtape Status -->
											<td class="p-3.5 whitespace-nowrap">
												{#if v.streamtapeStatus === 'active'}
													<span class="inline-flex items-center gap-1 rounded-full bg-emerald-500/15 px-2 py-0.5 text-[10px] font-bold text-emerald-400">
														<span class="h-1.5 w-1.5 rounded-full bg-emerald-400"></span> Active
													</span>
												{:else if v.streamtapeStatus === 'dead'}
													<span class="inline-flex items-center gap-1 rounded-full bg-destructive/15 px-2 py-0.5 text-[10px] font-bold text-destructive">
														<span class="h-1.5 w-1.5 rounded-full bg-destructive"></span> Dead
													</span>
												{:else}
													<span class="inline-flex items-center gap-1 rounded-full bg-amber-500/15 px-2 py-0.5 text-[10px] font-bold text-amber-400">
														<span class="h-1.5 w-1.5 rounded-full bg-amber-400"></span> Pending
													</span>
												{/if}
											</td>

											<!-- Last Refreshed -->
											<td class="p-3.5 whitespace-nowrap text-muted-foreground text-[11px]">
												{formatDate(v.lastRefreshedAt || v.createdAt)}
											</td>

											<!-- Actions -->
											<td class="p-3.5 text-right whitespace-nowrap">
												<div class="inline-flex items-center gap-1">
													<a
														href="/watch/{v.fileId || v.id}"
														target="_blank"
														class="rounded-lg p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
														title="Watch Video"
													>
														<ExternalLink class="h-4 w-4" />
													</a>
													<button
														type="button"
														disabled={deletingVideoId === v.id}
														onclick={() => deleteVideo(v.id, v.title)}
														class="rounded-lg p-1.5 text-destructive/80 hover:bg-destructive/10 hover:text-destructive transition-colors"
														title="Delete Video"
													>
														{#if deletingVideoId === v.id}
															<Loader2 class="h-4 w-4 animate-spin" />
														{:else}
															<Trash2 class="h-4 w-4" />
														{/if}
													</button>
												</div>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>

						<!-- Pagination Bar -->
						<div class="flex items-center justify-between border-t p-3 text-xs text-muted-foreground">
							<span>Showing page {videoPage} of {videosQuery.data.totalPages || 1} ({videosQuery.data.total} total)</span>
							<div class="flex items-center gap-2">
								<Button
									variant="outline"
									size="sm"
									disabled={videoPage <= 1}
									onclick={() => (videoPage = Math.max(1, videoPage - 1))}
									class="rounded-lg text-xs"
								>
									Previous
								</Button>
								<Button
									variant="outline"
									size="sm"
									disabled={videoPage >= (videosQuery.data.totalPages || 1)}
									onclick={() => (videoPage = videoPage + 1)}
									class="rounded-lg text-xs"
								>
									Next
								</Button>
							</div>
						</div>
					{/if}
				</div>
			</div>
		{/if}

		<!-- ================= TAB 3: USER MANAGEMENT ================= -->
		{#if activeTab === 'users'}
			<div class="space-y-4">
				<!-- Search -->
				<div class="flex items-center gap-3 rounded-2xl border border-border/80 bg-card p-4 shadow-sm">
					<div class="relative flex-1">
						<Search class="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
						<input
							type="text"
							value={userSearch}
							oninput={handleUserSearchInput}
							placeholder="Search users by username or email address..."
							class="w-full rounded-xl border border-border bg-background py-2 pr-4 pl-9 text-sm focus:ring-2 focus:ring-primary/20 focus:outline-none"
						/>
					</div>
				</div>

				<!-- Users Table -->
				<div class="overflow-hidden rounded-2xl border border-border/80 bg-card shadow-sm">
					{#if usersQuery.isPending}
						<div class="p-8 space-y-3">
							{#each Array(5) as _}
								<Skeleton class="h-14 w-full rounded-xl" />
							{/each}
						</div>
					{:else if usersQuery.data?.users?.length === 0}
						<div class="py-16 text-center">
							<Users class="mx-auto mb-3 h-10 w-10 text-muted-foreground opacity-40" />
							<p class="text-sm font-semibold text-foreground">No users found</p>
						</div>
					{:else if usersQuery.data}
						<div class="overflow-x-auto">
							<table class="w-full text-left text-xs">
								<thead class="border-b bg-muted/40 font-bold uppercase tracking-wider text-muted-foreground">
									<tr>
										<th class="p-3.5">User</th>
										<th class="p-3.5">Email</th>
										<th class="p-3.5">Role</th>
										<th class="p-3.5">18+ Preference</th>
										<th class="p-3.5">Uploads</th>
										<th class="p-3.5">Subscribers</th>
										<th class="p-3.5">Joined</th>
										<th class="p-3.5 text-right">Actions</th>
									</tr>
								</thead>
								<tbody class="divide-y divide-border/60">
									{#each usersQuery.data.users as u}
										<tr class="transition-colors hover:bg-muted/30">
											<!-- User Profile -->
											<td class="p-3.5 whitespace-nowrap">
												<div class="flex items-center gap-2.5">
													<img
														src={u.avatar || `https://api.dicebear.com/7.x/identicon/svg?seed=${u.username}`}
														alt=""
														class="h-7 w-7 rounded-full object-cover bg-muted"
													/>
													<span class="font-bold text-foreground">{u.username}</span>
												</div>
											</td>

											<!-- Email -->
											<td class="p-3.5 whitespace-nowrap text-muted-foreground font-mono">
												{u.email}
											</td>

											<!-- Admin Status -->
											<td class="p-3.5 whitespace-nowrap">
												{#if u.isAdmin}
													<span class="inline-flex items-center gap-1 rounded-full bg-amber-500/15 border border-amber-500/30 px-2.5 py-0.5 text-[10px] font-bold text-amber-500">
														<Crown class="h-3 w-3" /> Administrator
													</span>
												{:else}
													<span class="inline-flex items-center rounded-full bg-muted px-2.5 py-0.5 text-[10px] font-medium text-muted-foreground">
														Member
													</span>
												{/if}
											</td>

											<!-- 18+ Access Enabled -->
											<td class="p-3.5 whitespace-nowrap">
												{#if u.showAdultContent}
													<span class="inline-flex items-center gap-1 rounded-full bg-rose-500/15 px-2 py-0.5 text-[10px] font-bold text-rose-500">
														18+ Enabled
													</span>
												{:else}
													<span class="text-muted-foreground text-[11px]">Safe Mode</span>
												{/if}
											</td>

											<!-- Video Count -->
											<td class="p-3.5 whitespace-nowrap font-medium text-muted-foreground">
												{formatNumber(u.videoCount)}
											</td>

											<!-- Subscriber Count -->
											<td class="p-3.5 whitespace-nowrap font-medium text-muted-foreground">
												{formatNumber(u.subscriberCount)}
											</td>

											<!-- Joined Date -->
											<td class="p-3.5 whitespace-nowrap text-muted-foreground text-[11px]">
												{formatDate(u.createdAt)}
											</td>

											<!-- Actions -->
											<td class="p-3.5 text-right whitespace-nowrap">
												<div class="inline-flex items-center gap-1">
													<!-- Promote / Demote -->
													<Button
														variant="outline"
														size="sm"
														disabled={updatingUserId === u.id || u.id === userState.user?.id}
														onclick={() => toggleUserAdmin(u.id, u.isAdmin, u.username)}
														class="rounded-lg text-[10px] h-7 px-2 font-semibold {u.isAdmin ? 'text-destructive hover:bg-destructive/10' : 'text-amber-500 hover:bg-amber-500/10'}"
													>
														{#if updatingUserId === u.id}
															<Loader2 class="h-3 w-3 animate-spin" />
														{:else if u.isAdmin}
															Demote
														{:else}
															Make Admin
														{/if}
													</Button>

													<!-- View Profile Link -->
													<a
														href="/profile/{u.username}"
														target="_blank"
														class="rounded-lg p-1.5 text-muted-foreground hover:bg-muted hover:text-foreground transition-colors"
														title="View Profile"
													>
														<ExternalLink class="h-4 w-4" />
													</a>

													<!-- Delete User -->
													<button
														type="button"
														disabled={deletingUserId === u.id || u.id === userState.user?.id}
														onclick={() => deleteUser(u.id, u.username)}
														class="rounded-lg p-1.5 text-destructive/80 hover:bg-destructive/10 hover:text-destructive transition-colors disabled:opacity-30"
														title="Delete User Account"
													>
														{#if deletingUserId === u.id}
															<Loader2 class="h-4 w-4 animate-spin" />
														{:else}
															<Trash2 class="h-4 w-4" />
														{/if}
													</button>
												</div>
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>

						<!-- Pagination Bar -->
						<div class="flex items-center justify-between border-t p-3 text-xs text-muted-foreground">
							<span>Page {userPage} of {usersQuery.data.totalPages || 1} ({usersQuery.data.total} users)</span>
							<div class="flex items-center gap-2">
								<Button
									variant="outline"
									size="sm"
									disabled={userPage <= 1}
									onclick={() => (userPage = Math.max(1, userPage - 1))}
									class="rounded-lg text-xs"
								>
									Previous
								</Button>
								<Button
									variant="outline"
									size="sm"
									disabled={userPage >= (usersQuery.data.totalPages || 1)}
									onclick={() => (userPage = userPage + 1)}
									class="rounded-lg text-xs"
								>
									Next
								</Button>
							</div>
						</div>
					{/if}
				</div>
			</div>
		{/if}

		<!-- ================= TAB 4: OPERATIONS & LIFECYCLE ================= -->
		{#if activeTab === 'operations'}
			<div class="grid grid-cols-1 gap-6 lg:grid-cols-2">
				<!-- Streamtape Renewal Operations -->
				<div class="rounded-3xl border border-border/80 bg-card p-6 shadow-sm space-y-5">
					<div class="flex items-center gap-3">
						<div class="rounded-2xl bg-amber-500/15 p-2.5 text-amber-500">
							<RefreshCw class="h-6 w-6" />
						</div>
						<div>
							<h2 class="text-base font-bold text-foreground">Streamtape 28-Day Remote Renewal</h2>
							<p class="text-xs text-muted-foreground">
								Clones files upstream on Streamtape to reset their 30-day inactivity timer and preserve them indefinitely.
							</p>
						</div>
					</div>

					<div class="rounded-2xl border border-border/60 bg-muted/20 p-4 space-y-3">
						<div class="flex items-center justify-between text-xs font-semibold">
							<span>Expiration Threshold (Days)</span>
							<select
								bind:value={refreshDaysThreshold}
								class="rounded-lg border border-border bg-background px-2.5 py-1 text-xs"
							>
								<option value={7}>7 Days (Aggressive)</option>
								<option value={14}>14 Days (Standard)</option>
								<option value={28}>28 Days (Default)</option>
							</select>
						</div>

						<div class="flex items-center justify-between text-xs font-semibold">
							<span>Batch Size Limit</span>
							<select
								bind:value={refreshBatchLimit}
								class="rounded-lg border border-border bg-background px-2.5 py-1 text-xs"
							>
								<option value={10}>10 Videos</option>
								<option value={25}>25 Videos</option>
								<option value={50}>50 Videos</option>
							</select>
						</div>

						<Button
							onclick={runRefreshCycle}
							disabled={isRunningRefreshCycle}
							class="w-full rounded-xl bg-amber-600 hover:bg-amber-700 text-white font-bold text-xs"
						>
							{#if isRunningRefreshCycle}
								<Loader2 class="mr-2 h-4 w-4 animate-spin" />
								<span>Running Auto-Refresh Worker...</span>
							{:else}
								<RefreshCw class="mr-2 h-4 w-4" />
								<span>Run Renewal Cycle Now</span>
							{/if}
						</Button>
					</div>

					<!-- Execution Output Box -->
					<div class="space-y-1.5">
						<span class="text-xs font-bold text-muted-foreground uppercase tracking-wider">Operation Log</span>
						<div class="max-h-48 overflow-y-auto rounded-xl border border-border bg-black/80 p-3 font-mono text-[11px] text-zinc-300 space-y-1">
							{#if refreshCycleLogs.length === 0}
								<span class="text-zinc-600 italic">No cycle runs recorded in this session.</span>
							{:else}
								{#each refreshCycleLogs as logLine}
									<p class="leading-relaxed">{logLine}</p>
								{/each}
							{/if}
						</div>
					</div>
				</div>

				<!-- Cache & Diagnostic Controls -->
				<div class="rounded-3xl border border-border/80 bg-card p-6 shadow-sm space-y-5">
					<div class="flex items-center gap-3">
						<div class="rounded-2xl bg-blue-500/15 p-2.5 text-blue-500">
							<Server class="h-6 w-6" />
						</div>
						<div>
							<h2 class="text-base font-bold text-foreground">Cache & Infrastructure</h2>
							<p class="text-xs text-muted-foreground">
								Flush Valkey/Redis cache keys and view connected cloud microservice status.
							</p>
						</div>
					</div>

					<div class="space-y-3">
						<!-- Valkey Cache Flush -->
						<div class="flex items-center justify-between rounded-2xl border border-border/60 bg-muted/20 p-4">
							<div class="space-y-0.5">
								<p class="text-xs font-bold text-foreground">API Microcache (Valkey)</p>
								<p class="text-[11px] text-muted-foreground">Flushes search and feed query responses.</p>
							</div>
							<Button
								variant="outline"
								size="sm"
								onclick={flushApiCache}
								disabled={isFlushingCache}
								class="rounded-xl text-xs font-semibold"
							>
								{#if isFlushingCache}
									<Loader2 class="mr-1.5 h-3.5 w-3.5 animate-spin" />
								{:else}
									<Trash2 class="mr-1.5 h-3.5 w-3.5 text-destructive" />
								{/if}
								Flush Cache
							</Button>
						</div>

						<!-- OpenSearch Status -->
						<div class="flex items-center justify-between rounded-2xl border border-border/60 bg-muted/20 p-4">
							<div class="space-y-0.5">
								<p class="text-xs font-bold text-foreground">OpenSearch Search Cluster</p>
								<p class="text-[11px] text-muted-foreground">Fuzzy typo-tolerant full-text search indexing.</p>
							</div>
							<span class="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/15 px-3 py-1 text-xs font-bold text-emerald-400">
								<span class="h-2 w-2 rounded-full bg-emerald-400"></span> Active
							</span>
						</div>

						<!-- PostgreSQL Database Status -->
						<div class="flex items-center justify-between rounded-2xl border border-border/60 bg-muted/20 p-4">
							<div class="space-y-0.5">
								<p class="text-xs font-bold text-foreground">PostgreSQL (Aiven Cloud)</p>
								<p class="text-[11px] text-muted-foreground">Trigram fuzzy index + relational persistence.</p>
							</div>
							<span class="inline-flex items-center gap-1.5 rounded-full bg-emerald-500/15 px-3 py-1 text-xs font-bold text-emerald-400">
								<span class="h-2 w-2 rounded-full bg-emerald-400"></span> Connected
							</span>
						</div>
					</div>
				</div>
			</div>
		{/if}

		<!-- ================= TAB 5: KILL SWITCHES & SUPERPOWERS ================= -->
		{#if activeTab === 'killswitches'}
			<div class="space-y-6">
				<!-- Threat & Posture Header Banner -->
				<div
					class="relative overflow-hidden rounded-3xl border p-6 shadow-xl transition-all duration-300 md:p-8 {countActiveKillSwitches(killSwitchesQuery.data) > 0 ? 'border-destructive/40 bg-gradient-to-r from-destructive/15 via-background to-amber-500/10' : 'border-emerald-500/30 bg-gradient-to-r from-emerald-500/10 via-background to-cyan-500/10'}"
				>
					<div class="flex flex-col gap-5 md:flex-row md:items-center md:justify-between">
						<div class="space-y-2">
							<div
								class="inline-flex items-center gap-2 rounded-full border px-3 py-1 text-xs font-black tracking-wider uppercase {countActiveKillSwitches(killSwitchesQuery.data) > 0 ? 'border-destructive/40 bg-destructive/20 text-destructive animate-pulse' : 'border-emerald-500/40 bg-emerald-500/15 text-emerald-400'}"
							>
								<Zap class="h-3.5 w-3.5" />
								<span>{countActiveKillSwitches(killSwitchesQuery.data) > 0 ? 'Emergency Lockdown State' : 'Normal Operational Posture'}</span>
							</div>
							<h2 class="text-2xl font-black tracking-tight text-foreground md:text-3xl">
								System Kill Switches & Superpowers
							</h2>
							<p class="max-w-2xl text-xs text-muted-foreground leading-relaxed md:text-sm">
								Zero-latency Valkey/Redis emergency enforcement synchronized with durable PostgreSQL persistence. Administrators automatically bypass all restrictions.
							</p>
						</div>

						<!-- Quick Bulk Action Controls -->
						<div class="flex flex-wrap items-center gap-2.5">
							<Button
								variant="outline"
								size="sm"
								disabled={isBulkOperating || countActiveKillSwitches(killSwitchesQuery.data) === 0}
								onclick={() => bulkSetKillSwitches(false)}
								class="rounded-xl border-emerald-500/40 text-emerald-400 hover:bg-emerald-500/15 text-xs font-bold gap-2"
							>
								{#if isBulkOperating}
									<Loader2 class="h-3.5 w-3.5 animate-spin" />
								{:else}
									<CheckCircle2 class="h-3.5 w-3.5" />
								{/if}
								<span>Disarm All (Normalize)</span>
							</Button>
							<Button
								variant="destructive"
								size="sm"
								disabled={isBulkOperating}
								onclick={() => bulkSetKillSwitches(true)}
								class="rounded-xl text-xs font-bold gap-2 shadow-lg shadow-destructive/20"
							>
								{#if isBulkOperating}
									<Loader2 class="h-3.5 w-3.5 animate-spin" />
								{:else}
									<AlertOctagon class="h-3.5 w-3.5" />
								{/if}
								<span>Emergency Total Lockdown</span>
							</Button>
						</div>
					</div>
				</div>

				<!-- Kill Switch Cards Grid -->
				{#if killSwitchesQuery.isPending}
					<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
						{#each Array(4) as _}
							<Skeleton class="h-64 rounded-3xl" />
						{/each}
					</div>
				{:else if killSwitchesQuery.isError}
					<div class="rounded-3xl border border-destructive/30 bg-destructive/10 p-6 text-center">
						<AlertTriangle class="mx-auto h-8 w-8 text-destructive mb-2" />
						<p class="text-sm font-bold text-destructive">Failed to load system kill switches</p>
						<Button
							variant="outline"
							size="sm"
							class="mt-3 rounded-full text-xs"
							onclick={() => killSwitchesQuery.refetch()}
						>
							Retry
						</Button>
					</div>
				{:else}
					<div class="grid grid-cols-1 gap-6 md:grid-cols-2">
						<!-- Switch 1: Platform Maintenance Mode -->
						<div
							class="relative overflow-hidden rounded-3xl border p-6 shadow-md transition-all duration-300 {isMaintenanceActive ? 'border-destructive/60 bg-destructive/10 shadow-destructive/10' : 'border-border/80 bg-card'}"
						>
							<div class="flex items-start justify-between gap-4">
								<div class="flex items-center gap-3.5">
									<div class="rounded-2xl p-3 {isMaintenanceActive ? 'bg-destructive/20 text-destructive' : 'bg-muted text-muted-foreground'}">
										<Power class="h-6 w-6" />
									</div>
									<div>
										<div class="flex items-center gap-2">
											<h3 class="text-base font-bold text-foreground">Platform Maintenance Mode</h3>
											<span class="rounded-full px-2 py-0.5 text-[10px] font-black uppercase tracking-wider {isMaintenanceActive ? 'bg-destructive/30 text-destructive border border-destructive/50 animate-pulse' : 'bg-muted text-muted-foreground'}">
												{isMaintenanceActive ? 'Armed (503)' : 'Standby'}
											</span>
										</div>
										<p class="text-xs text-muted-foreground">Key: <code class="text-[11px] font-mono text-primary font-bold">maintenance_mode</code></p>
									</div>
								</div>

								<!-- Custom Animated Switch Button -->
								<button
									type="button"
									role="switch"
									aria-checked={isMaintenanceActive}
									disabled={togglingKey === 'maintenance_mode'}
									onclick={() => toggleKillSwitch('maintenance_mode', isMaintenanceActive)}
									class="relative inline-flex h-8 w-16 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-destructive focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 {isMaintenanceActive ? 'bg-destructive shadow-md shadow-destructive/40' : 'bg-muted-foreground/30'}"
								>
									{#if togglingKey === 'maintenance_mode'}
										<Loader2 class="mx-auto h-4 w-4 animate-spin text-foreground" />
									{:else}
										<span
											class="pointer-events-none inline-block h-6 w-6 transform rounded-full bg-background shadow-lg ring-0 transition duration-200 ease-in-out {isMaintenanceActive ? 'translate-x-8 bg-destructive-foreground' : 'translate-x-1'}"
										></span>
									{/if}
								</button>
							</div>

							<p class="mt-4 text-xs leading-relaxed text-muted-foreground">
								Halts all non-admin mutation requests (POST/PUT/DELETE/PATCH) across the entire platform with an instant HTTP 503 Service Unavailable. Admin accounts retain full unrestricted access.
							</p>

							<div class="mt-5 space-y-2 rounded-2xl border border-border/50 bg-background/50 p-3.5 text-[11px]">
								<div class="flex items-center justify-between">
									<span class="text-muted-foreground font-medium">Impact Severity:</span>
									<span class="font-bold text-destructive">CRITICAL • Platform-wide</span>
								</div>
								<div class="flex items-center justify-between">
									<span class="text-muted-foreground font-medium">Enforcement Layer:</span>
									<span class="font-mono text-emerald-400 font-semibold">Valkey Microcache &lt;1ms</span>
								</div>
								<div class="flex items-center justify-between">
									<span class="text-muted-foreground font-medium">Last Modified:</span>
									<span class="text-foreground font-semibold">{formatDateTime(maintenance?.updated_at || maintenance?.updatedAt)} by {maintenance?.updated_by || maintenance?.updatedBy || 'system'}</span>
								</div>
							</div>
						</div>

						<!-- Switch 2: Emergency Upload Freeze -->
						<div
							class="relative overflow-hidden rounded-3xl border p-6 shadow-md transition-all duration-300 {isUploadsActive ? 'border-amber-500/60 bg-amber-500/10 shadow-amber-500/10' : 'border-border/80 bg-card'}"
						>
							<div class="flex items-start justify-between gap-4">
								<div class="flex items-center gap-3.5">
									<div class="rounded-2xl p-3 {isUploadsActive ? 'bg-amber-500/20 text-amber-500' : 'bg-muted text-muted-foreground'}">
										<Upload class="h-6 w-6" />
									</div>
									<div>
										<div class="flex items-center gap-2">
											<h3 class="text-base font-bold text-foreground">Emergency Upload Freeze</h3>
											<span class="rounded-full px-2 py-0.5 text-[10px] font-black uppercase tracking-wider {isUploadsActive ? 'bg-amber-500/30 text-amber-500 border border-amber-500/50 animate-pulse' : 'bg-muted text-muted-foreground'}">
												{isUploadsActive ? 'Frozen' : 'Normal'}
											</span>
										</div>
										<p class="text-xs text-muted-foreground">Key: <code class="text-[11px] font-mono text-primary font-bold">freeze_uploads</code></p>
									</div>
								</div>

								<!-- Switch Button -->
								<button
									type="button"
									role="switch"
									aria-checked={isUploadsActive}
									disabled={togglingKey === 'freeze_uploads'}
									onclick={() => toggleKillSwitch('freeze_uploads', isUploadsActive)}
									class="relative inline-flex h-8 w-16 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-amber-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 {isUploadsActive ? 'bg-amber-500 shadow-md shadow-amber-500/40' : 'bg-muted-foreground/30'}"
								>
									{#if togglingKey === 'freeze_uploads'}
										<Loader2 class="mx-auto h-4 w-4 animate-spin text-foreground" />
									{:else}
										<span
											class="pointer-events-none inline-block h-6 w-6 transform rounded-full bg-background shadow-lg ring-0 transition duration-200 ease-in-out {isUploadsActive ? 'translate-x-8 bg-amber-950' : 'translate-x-1'}"
										></span>
									{/if}
								</button>
							</div>

							<p class="mt-4 text-xs leading-relaxed text-muted-foreground">
								Blocks all chunk-based video uploads, remote URL upload ingestions, and background upload workers immediately. Protects disk space and Streamtape quotas during floods.
							</p>

							<div class="mt-5 space-y-2 rounded-2xl border border-border/50 bg-background/50 p-3.5 text-[11px]">
								<div class="flex items-center justify-between">
									<span class="text-muted-foreground font-medium">Impact Severity:</span>
									<span class="font-bold text-amber-400">HIGH • Storage & Ingestion</span>
								</div>
								<div class="flex items-center justify-between">
									<span class="text-muted-foreground font-medium">Enforcement Layer:</span>
									<span class="font-mono text-emerald-400 font-semibold">Valkey Microcache &lt;1ms</span>
								</div>
								<div class="flex items-center justify-between">
									<span class="text-muted-foreground font-medium">Last Modified:</span>
									<span class="text-foreground font-semibold">{formatDateTime(uploads?.updated_at || uploads?.updatedAt)} by {uploads?.updated_by || uploads?.updatedBy || 'system'}</span>
								</div>
							</div>
						</div>

						<!-- Switch: Emergency Playback Freeze (Content Blackout) -->
						<div
							class="relative overflow-hidden rounded-3xl border p-6 shadow-md transition-all duration-300 {isStreamingActive ? 'border-rose-600/60 bg-rose-600/10 shadow-rose-600/10' : 'border-border/80 bg-card'}"
						>
							<div class="flex items-start justify-between gap-4">
								<div class="flex items-center gap-3.5">
									<div class="rounded-2xl p-3 {isStreamingActive ? 'bg-rose-600/20 text-rose-500' : 'bg-muted text-muted-foreground'}">
										<Film class="h-6 w-6" />
									</div>
									<div>
										<div class="flex items-center gap-2">
											<h3 class="text-base font-bold text-foreground">Emergency Playback Freeze</h3>
											<span class="rounded-full px-2 py-0.5 text-[10px] font-black uppercase tracking-wider {isStreamingActive ? 'bg-rose-600/30 text-rose-400 border border-rose-600/50 animate-pulse' : 'bg-muted text-muted-foreground'}">
												{isStreamingActive ? 'Blackout Active' : 'Normal'}
											</span>
										</div>
										<p class="text-xs text-muted-foreground">Key: <code class="text-[11px] font-mono text-primary font-bold">freeze_streaming</code></p>
									</div>
								</div>

								<!-- Switch Button -->
								<button
									type="button"
									role="switch"
									aria-checked={isStreamingActive}
									disabled={togglingKey === 'freeze_streaming'}
									onclick={() => toggleKillSwitch('freeze_streaming', isStreamingActive)}
									class="relative inline-flex h-8 w-16 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-rose-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 {isStreamingActive ? 'bg-rose-600 shadow-md shadow-rose-600/40' : 'bg-muted-foreground/30'}"
								>
									{#if togglingKey === 'freeze_streaming'}
										<Loader2 class="mx-auto h-4 w-4 animate-spin text-foreground" />
									{:else}
										<span
											class="pointer-events-none inline-block h-6 w-6 transform rounded-full bg-background shadow-lg ring-0 transition duration-200 ease-in-out {isStreamingActive ? 'translate-x-8 bg-rose-950' : 'translate-x-1'}"
										></span>
									{/if}
								</button>
							</div>

							<p class="mt-4 text-xs leading-relaxed text-muted-foreground">
								Instantly halts all video stream resolution and embedded player delivery platform-wide. Halts CDN bandwidth and prevents viewing during critical emergencies. Admins retain full playback bypass.
							</p>

							<div class="mt-5 space-y-2 rounded-2xl border border-border/50 bg-background/50 p-3.5 text-[11px]">
								<div class="flex items-center justify-between">
									<span class="text-muted-foreground font-medium">Impact Severity:</span>
									<span class="font-bold text-rose-500">CRITICAL • Playback & CDN Stream</span>
								</div>
								<div class="flex items-center justify-between">
									<span class="text-muted-foreground font-medium">Enforcement Layer:</span>
									<span class="font-mono text-emerald-400 font-semibold">Valkey Microcache &lt;1ms</span>
								</div>
								<div class="flex items-center justify-between">
									<span class="text-muted-foreground font-medium">Last Modified:</span>
									<span class="text-foreground font-semibold">{formatDateTime(streaming?.updated_at || streaming?.updatedAt)} by {streaming?.updated_by || streaming?.updatedBy || 'system'}</span>
								</div>
							</div>
						</div>

						<!-- Switch 3: Registration Lockdown -->
						<div
							class="relative overflow-hidden rounded-3xl border p-6 shadow-md transition-all duration-300 {isRegistrationActive ? 'border-cyan-500/60 bg-cyan-500/10 shadow-cyan-500/10' : 'border-border/80 bg-card'}"
						>
							<div class="flex items-start justify-between gap-4">
								<div class="flex items-center gap-3.5">
									<div class="rounded-2xl p-3 {isRegistrationActive ? 'bg-cyan-500/20 text-cyan-500' : 'bg-muted text-muted-foreground'}">
										<Lock class="h-6 w-6" />
									</div>
									<div>
										<div class="flex items-center gap-2">
											<h3 class="text-base font-bold text-foreground">Registration Lockdown</h3>
											<span class="rounded-full px-2 py-0.5 text-[10px] font-black uppercase tracking-wider {isRegistrationActive ? 'bg-cyan-500/30 text-cyan-400 border border-cyan-500/50 animate-pulse' : 'bg-muted text-muted-foreground'}">
												{isRegistrationActive ? 'Locked' : 'Open'}
											</span>
										</div>
										<p class="text-xs text-muted-foreground">Key: <code class="text-[11px] font-mono text-primary font-bold">freeze_registration</code></p>
									</div>
								</div>

								<!-- Switch Button -->
								<button
									type="button"
									role="switch"
									aria-checked={isRegistrationActive}
									disabled={togglingKey === 'freeze_registration'}
									onclick={() => toggleKillSwitch('freeze_registration', isRegistrationActive)}
									class="relative inline-flex h-8 w-16 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-cyan-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 {isRegistrationActive ? 'bg-cyan-500 shadow-md shadow-cyan-500/40' : 'bg-muted-foreground/30'}"
								>
									{#if togglingKey === 'freeze_registration'}
										<Loader2 class="mx-auto h-4 w-4 animate-spin text-foreground" />
									{:else}
										<span
											class="pointer-events-none inline-block h-6 w-6 transform rounded-full bg-background shadow-lg ring-0 transition duration-200 ease-in-out {isRegistrationActive ? 'translate-x-8 bg-cyan-950' : 'translate-x-1'}"
										></span>
									{/if}
								</button>
							</div>

							<p class="mt-4 text-xs leading-relaxed text-muted-foreground">
								Closes new account registration routes instantly. Existing accounts can continue to log in and authenticate normally without disruption.
							</p>

							<div class="mt-5 space-y-2 rounded-2xl border border-border/50 bg-background/50 p-3.5 text-[11px]">
								<div class="flex items-center justify-between">
									<span class="text-muted-foreground font-medium">Impact Severity:</span>
									<span class="font-bold text-cyan-400">MODERATE • User Onboarding</span>
								</div>
								<div class="flex items-center justify-between">
									<span class="text-muted-foreground font-mono text-emerald-400 font-semibold">Valkey Microcache &lt;1ms</span>
								</div>
								<div class="flex items-center justify-between">
									<span class="text-muted-foreground font-medium">Last Modified:</span>
									<span class="text-foreground font-semibold">{formatDateTime(reg?.updated_at || reg?.updatedAt)} by {reg?.updated_by || reg?.updatedBy || 'system'}</span>
								</div>
							</div>
						</div>

						<!-- Switch 4: Commenting Freeze -->
						<div
							class="relative overflow-hidden rounded-3xl border p-6 shadow-md transition-all duration-300 {isCommentsActive ? 'border-purple-500/60 bg-purple-500/10 shadow-purple-500/10' : 'border-border/80 bg-card'}"
						>
							<div class="flex items-start justify-between gap-4">
								<div class="flex items-center gap-3.5">
									<div class="rounded-2xl p-3 {isCommentsActive ? 'bg-purple-500/20 text-purple-400' : 'bg-muted text-muted-foreground'}">
										<MessageSquare class="h-6 w-6" />
									</div>
									<div>
										<div class="flex items-center gap-2">
											<h3 class="text-base font-bold text-foreground">Commenting Freeze</h3>
											<span class="rounded-full px-2 py-0.5 text-[10px] font-black uppercase tracking-wider {isCommentsActive ? 'bg-purple-500/30 text-purple-400 border border-purple-500/50 animate-pulse' : 'bg-muted text-muted-foreground'}">
												{isCommentsActive ? 'Frozen' : 'Active'}
											</span>
										</div>
										<p class="text-xs text-muted-foreground">Key: <code class="text-[11px] font-mono text-primary font-bold">freeze_comments</code></p>
									</div>
								</div>

								<!-- Switch Button -->
								<button
									type="button"
									role="switch"
									aria-checked={isCommentsActive}
									disabled={togglingKey === 'freeze_comments'}
									onclick={() => toggleKillSwitch('freeze_comments', isCommentsActive)}
									class="relative inline-flex h-8 w-16 shrink-0 cursor-pointer items-center rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-purple-500 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 {isCommentsActive ? 'bg-purple-500 shadow-md shadow-purple-500/40' : 'bg-muted-foreground/30'}"
								>
									{#if togglingKey === 'freeze_comments'}
										<Loader2 class="mx-auto h-4 w-4 animate-spin text-foreground" />
									{:else}
										<span
											class="pointer-events-none inline-block h-6 w-6 transform rounded-full bg-background shadow-lg ring-0 transition duration-200 ease-in-out {isCommentsActive ? 'translate-x-8 bg-purple-950' : 'translate-x-1'}"
										></span>
									{/if}
								</button>
							</div>

							<p class="mt-4 text-xs leading-relaxed text-muted-foreground">
								Temporarily disables adding new comments or replies across all videos. Existing comments remain visible. Used to neutralize bot raids or targeted harassment spikes.
							</p>

							<div class="mt-5 space-y-2 rounded-2xl border border-border/50 bg-background/50 p-3.5 text-[11px]">
								<div class="flex items-center justify-between">
									<span class="text-muted-foreground font-medium">Impact Severity:</span>
									<span class="font-bold text-purple-400">MODERATE • Community Interactivity</span>
								</div>
								<div class="flex items-center justify-between">
									<span class="text-muted-foreground font-medium">Enforcement Layer:</span>
									<span class="font-mono text-emerald-400 font-semibold">Valkey Microcache &lt;1ms</span>
								</div>
								<div class="flex items-center justify-between">
									<span class="text-muted-foreground font-medium">Last Modified:</span>
									<span class="text-foreground font-semibold">{formatDateTime(comments?.updated_at || comments?.updatedAt)} by {comments?.updated_by || comments?.updatedBy || 'system'}</span>
								</div>
							</div>
						</div>
					</div>
				{/if}
			</div>
		{/if}
	</div>
{/if}
