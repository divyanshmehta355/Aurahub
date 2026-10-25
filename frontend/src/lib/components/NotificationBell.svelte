<script lang="ts">
	import { onMount } from 'svelte';
	import { fetchApi, API_URL } from '#lib/api';
	import { userState } from '#lib/user.svelte';
	import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';
	import {
		Bell,
		Heart,
		MessageSquare,
		Reply,
		Video,
		CheckCheck,
		Trash2,
		X,
		Loader2,
		Sparkles
	} from 'lucide-svelte';
	import { Button } from '#lib/components/ui/button';

	const queryClient = useQueryClient();

	let isOpen = $state(false);
	let activeFilter = $state<'all' | 'unread'>('all');
	let dropdownRef: HTMLDivElement | null = $state(null);

	// Real-time toast state for new incoming notifications
	let toastNotification = $state<any>(null);
	let toastTimeout: any = null;

	function showRealtimeToast(notif: any) {
		toastNotification = notif;
		if (toastTimeout) clearTimeout(toastTimeout);
		toastTimeout = setTimeout(() => {
			toastNotification = null;
		}, 5000);
	}

	const notificationsQuery = createQuery(
		() => ({
			queryKey: ['notifications'],
			queryFn: async () => {
				if (!userState.user) return { notifications: [], unreadCount: 0 };
				const res = await fetchApi('/notifications?limit=30').catch(() => ({
					notifications: [],
					unreadCount: 0
				}));
				return res || { notifications: [], unreadCount: 0 };
			},
			enabled: !!userState.user,
			// Real-time updates handled by Valkey SSE stream; use 2 min gentle fallback
			refetchInterval: 120000
		}),
		() => queryClient
	);

	// Valkey-powered Real-time Server-Sent Events (SSE) Stream
	$effect(() => {
		if (!userState.user || typeof window === 'undefined') return;

		const streamUrl = `${API_URL}/notifications/stream`;
		const es = new EventSource(streamUrl, { withCredentials: true });

		es.addEventListener('init', (e: MessageEvent) => {
			try {
				const data = JSON.parse(e.data);
				if (typeof data.unreadCount === 'number') {
					queryClient.setQueryData(['notifications'], (old: any) => ({
						notifications: old?.notifications || [],
						unreadCount: data.unreadCount
					}));
				}
			} catch (err) {
				console.error('Failed to parse SSE init event:', err);
			}
		});

		es.addEventListener('notification', (e: MessageEvent) => {
			try {
				const data = JSON.parse(e.data);
				if (data.event === 'new_notification' && data.notification) {
					queryClient.setQueryData(['notifications'], (old: any) => {
						if (!old) {
							return { notifications: [data.notification], unreadCount: data.unreadCount ?? 1 };
						}
						const exists = old.notifications.some((n: any) => n.id === data.notification.id);
						if (exists) return old;
						return {
							notifications: [data.notification, ...old.notifications],
							unreadCount: data.unreadCount ?? ((old.unreadCount || 0) + 1)
						};
					});
					showRealtimeToast(data.notification);
				} else if (data.event === 'all_read') {
					queryClient.setQueryData(['notifications'], (old: any) => {
						if (!old) return old;
						return {
							...old,
							notifications: old.notifications.map((n: any) => ({ ...n, isRead: true })),
							unreadCount: 0
						};
					});
				} else if (data.event === 'notification_read') {
					queryClient.setQueryData(['notifications'], (old: any) => {
						if (!old) return old;
						return {
							...old,
							notifications: old.notifications.map((n: any) =>
								n.id === data.id ? { ...n, isRead: true } : n
							),
							unreadCount: data.unreadCount ?? Math.max(0, (old.unreadCount || 0) - 1)
						};
					});
				} else if (data.event === 'notification_deleted') {
					queryClient.setQueryData(['notifications'], (old: any) => {
						if (!old) return old;
						return {
							...old,
							notifications: old.notifications.filter((n: any) => n.id !== data.id),
							unreadCount: data.unreadCount ?? old.unreadCount
						};
					});
				} else if (data.event === 'all_cleared') {
					queryClient.setQueryData(['notifications'], {
						notifications: [],
						unreadCount: 0
					});
				} else if (data.event === 'unread_count_updated') {
					queryClient.setQueryData(['notifications'], (old: any) => ({
						notifications: old?.notifications || [],
						unreadCount: data.unreadCount
					}));
				} else if (data.event === 'batch_upload_progress') {
					window.dispatchEvent(new CustomEvent('batch_upload_progress', { detail: data }));
				}
			} catch (err) {
				console.error('Failed to parse real-time notification event:', err);
			}
		});

		es.onerror = () => {
			// EventSource will automatically attempt reconnection
		};

		return () => {
			es.close();
		};
	});

	const markAllReadMutation = createMutation(
		() => ({
			mutationFn: async () => {
				return await fetchApi('/notifications/read-all', { method: 'POST' });
			},
			onSuccess: () => {
				queryClient.invalidateQueries({ queryKey: ['notifications'] });
			}
		}),
		() => queryClient
	);

	const markSingleReadMutation = createMutation(
		() => ({
			mutationFn: async (id: string) => {
				return await fetchApi(`/notifications/${id}/read`, { method: 'POST' });
			},
			onSuccess: () => {
				queryClient.invalidateQueries({ queryKey: ['notifications'] });
			}
		}),
		() => queryClient
	);

	const deleteSingleMutation = createMutation(
		() => ({
			mutationFn: async (id: string) => {
				return await fetchApi(`/notifications/${id}`, { method: 'DELETE' });
			},
			onSuccess: () => {
				queryClient.invalidateQueries({ queryKey: ['notifications'] });
			}
		}),
		() => queryClient
	);

	const clearAllMutation = createMutation(
		() => ({
			mutationFn: async () => {
				return await fetchApi('/notifications', { method: 'DELETE' });
			},
			onSuccess: () => {
				queryClient.invalidateQueries({ queryKey: ['notifications'] });
			}
		}),
		() => queryClient
	);

	function toggleDropdown() {
		isOpen = !isOpen;
	}

	function closeDropdown() {
		isOpen = false;
	}

	function handleClickOutside(event: MouseEvent) {
		if (isOpen && dropdownRef && !dropdownRef.contains(event.target as Node)) {
			closeDropdown();
		}
	}

	onMount(() => {
		document.addEventListener('click', handleClickOutside);
		return () => {
			document.removeEventListener('click', handleClickOutside);
		};
	});

	function handleNotificationClick(n: any) {
		if (!n.isRead) {
			markSingleReadMutation.mutate(n.id);
		}
		closeDropdown();
		toastNotification = null;
		if (n.videoId) {
			window.location.href = `/watch/${n.videoId}`;
		}
	}

	function handleDeleteNotification(e: Event, id: string) {
		e.stopPropagation();
		deleteSingleMutation.mutate(id);
	}

	function handleMarkAllRead() {
		markAllReadMutation.mutate();
	}

	function handleClearAll() {
		if (!confirm('Clear all notifications?')) return;
		clearAllMutation.mutate();
	}

	function formatTimeAgo(dateString: string): string {
		if (!dateString) return '';
		const now = new Date();
		const date = new Date(dateString);
		const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);
		if (seconds < 60) return 'Just now';
		const minutes = Math.floor(seconds / 60);
		if (minutes < 60) return `${minutes}m ago`;
		const hours = Math.floor(minutes / 60);
		if (hours < 24) return `${hours}h ago`;
		const days = Math.floor(hours / 24);
		if (days < 30) return `${days}d ago`;
		return date.toLocaleDateString();
	}

	const notifications = $derived(notificationsQuery.data?.notifications || []);
	const unreadCount = $derived(notificationsQuery.data?.unreadCount || 0);
	const filteredNotifications = $derived(
		activeFilter === 'unread' ? notifications.filter((n: any) => !n.isRead) : notifications
	);
</script>

<!-- Real-time Toast Banner -->
{#if toastNotification}
	<div
		class="fixed top-20 right-4 z-[9999] max-w-sm w-full bg-card/95 backdrop-blur-md text-card-foreground border border-primary/30 rounded-2xl p-3.5 shadow-2xl flex items-start gap-3 animate-in fade-in slide-in-from-top-4 duration-300"
	>
		<!-- Avatar + Type Badge -->
		<div class="relative shrink-0">
			<img
				src={toastNotification.sender?.avatar ||
					`https://api.dicebear.com/7.x/identicon/svg?seed=${toastNotification.sender?.username || 'user'}`}
				alt={toastNotification.sender?.username || 'User'}
				class="w-10 h-10 rounded-full border border-border object-cover bg-muted"
			/>
			<div
				class="absolute -bottom-1 -right-1 p-0.5 rounded-full text-white shadow-sm {toastNotification.type ===
				'like'
					? 'bg-red-500'
					: toastNotification.type === 'comment'
						? 'bg-blue-500'
						: toastNotification.type === 'reply'
							? 'bg-indigo-500'
							: 'bg-primary'}"
			>
				{#if toastNotification.type === 'like'}
					<Heart class="w-3 h-3 fill-current" />
				{:else if toastNotification.type === 'comment'}
					<MessageSquare class="w-3 h-3 fill-current" />
				{:else if toastNotification.type === 'reply'}
					<Reply class="w-3 h-3" />
				{:else}
					<Video class="w-3 h-3" />
				{/if}
			</div>
		</div>

		<!-- Content -->
		<button
			type="button"
			class="flex-1 min-w-0 cursor-pointer text-left bg-transparent border-0 p-0"
			onclick={() => handleNotificationClick(toastNotification)}
		>
			<p class="text-xs font-semibold text-foreground truncate">
				@{toastNotification.sender?.username || 'Someone'}
				<span class="font-normal text-muted-foreground ml-1">
					{#if toastNotification.type === 'like'}
						liked your video
					{:else if toastNotification.type === 'comment'}
						commented on your video
					{:else if toastNotification.type === 'reply'}
						replied to your comment
					{:else if toastNotification.type === 'new_video'}
						uploaded a new video
					{:else}
						sent a notification
					{/if}
				</span>
			</p>
			{#if toastNotification.comment?.text}
				<p class="text-xs text-muted-foreground line-clamp-1 mt-0.5 italic">
					"{toastNotification.comment.text}"
				</p>
			{:else if toastNotification.video?.title}
				<p class="text-xs text-muted-foreground line-clamp-1 mt-0.5">
					{toastNotification.video.title}
				</p>
			{/if}
			<span class="text-[10px] text-primary font-medium mt-1 inline-block">Just now &bull; Click to view</span>
		</button>

		<!-- Dismiss button -->
		<button
			type="button"
			onclick={() => (toastNotification = null)}
			class="text-muted-foreground hover:text-foreground p-1 rounded-full hover:bg-muted shrink-0"
			aria-label="Dismiss toast"
		>
			<X class="w-4 h-4" />
		</button>
	</div>
{/if}

<div class="relative" bind:this={dropdownRef}>
	<!-- Bell Button -->
	<button
		type="button"
		onclick={toggleDropdown}
		class="relative rounded-full p-2 text-muted-silver transition-colors hover:bg-deep-slate hover:text-soft-ivory focus:outline-none"
		aria-label="Notifications"
		title="Notifications"
	>
		<Bell class="h-5 w-5" />
		{#if unreadCount > 0}
			<span
				class="absolute top-1 right-1 flex h-4 min-w-[16px] items-center justify-center rounded-full bg-aura-lavender px-1 text-[10px] font-bold text-midnight-ink shadow-sm animate-in zoom-in-75"
			>
				{unreadCount > 9 ? '9+' : unreadCount}
			</span>
		{/if}
	</button>

	<!-- Dropdown Popover -->
	{#if isOpen}
		<div
			class="absolute right-0 mt-2 w-80 sm:w-96 rounded-3xl border border-slate-border/80 bg-deep-slate/95 text-soft-ivory shadow-2xl z-50 overflow-hidden backdrop-blur-2xl animate-in fade-in slide-in-from-top-2 duration-200"
		>
			<!-- Header -->
			<div class="flex items-center justify-between border-b border-slate-border/60 px-4 py-3 bg-midnight-ink/40">
				<div class="flex items-center gap-2">
					<h3 class="font-bold text-sm text-soft-ivory">Notifications</h3>
					{#if unreadCount > 0}
						<span class="rounded-full bg-aura-lavender/20 px-2 py-0.5 text-[10px] font-bold text-aura-lavender">
							{unreadCount} new
						</span>
					{/if}
				</div>

				<div class="flex items-center gap-1">
					{#if unreadCount > 0}
						<button
							type="button"
							onclick={handleMarkAllRead}
							disabled={markAllReadMutation.isPending}
							class="text-muted-silver hover:text-aura-lavender hover:bg-slate-elevated rounded-full p-1.5 text-xs transition-colors"
							title="Mark all as read"
						>
							<CheckCheck class="h-4 w-4" />
						</button>
					{/if}
					{#if notifications.length > 0}
						<button
							type="button"
							onclick={handleClearAll}
							disabled={clearAllMutation.isPending}
							class="text-muted-silver hover:text-destructive hover:bg-destructive/10 rounded-full p-1.5 text-xs transition-colors"
							title="Clear all notifications"
						>
							<Trash2 class="h-4 w-4" />
						</button>
					{/if}
				</div>
			</div>

			<!-- Filter Tabs -->
			<div class="flex border-b border-slate-border/60 px-4 bg-midnight-ink/20 text-xs font-medium">
				<button
					type="button"
					onclick={() => (activeFilter = 'all')}
					class="py-2.5 px-3 border-b-2 transition-colors {activeFilter === 'all'
						? 'border-aura-lavender text-aura-lavender font-bold'
						: 'border-transparent text-muted-silver hover:text-soft-ivory'}"
				>
					All ({notifications.length})
				</button>
				<button
					type="button"
					onclick={() => (activeFilter = 'unread')}
					class="py-2.5 px-3 border-b-2 transition-colors {activeFilter === 'unread'
						? 'border-aura-lavender text-aura-lavender font-bold'
						: 'border-transparent text-muted-silver hover:text-soft-ivory'}"
				>
					Unread ({unreadCount})
				</button>
			</div>

			<!-- Notifications List -->
			<div class="max-h-[420px] overflow-y-auto divide-y divide-slate-border/40">
				{#if notificationsQuery.isPending}
					<div class="flex flex-col items-center justify-center p-8 text-muted-silver gap-2">
						<Loader2 class="h-6 w-6 animate-spin text-aura-lavender" />
						<span class="text-xs">Loading notifications...</span>
					</div>
				{:else if filteredNotifications.length === 0}
					<div class="flex flex-col items-center justify-center py-12 px-4 text-center">
						<div class="rounded-2xl bg-slate-elevated p-3.5 mb-3 text-muted-silver">
							<Sparkles class="h-6 w-6 text-aura-lavender/70" />
						</div>
						<p class="text-sm font-bold text-soft-ivory">
							{activeFilter === 'unread' ? 'No unread notifications' : 'All caught up!'}
						</p>
						<p class="text-xs text-muted-silver mt-1 max-w-[200px]">
							{activeFilter === 'unread'
								? 'You have read all your recent notifications.'
								: 'Activities on your videos and comments will appear here in real time.'}
						</p>
					</div>
				{:else}
					{#each filteredNotifications as n (n.id)}
						<div
							role="button"
							tabindex="0"
							onkeydown={(e) => {
								if (e.key === 'Enter' || e.key === ' ') {
									e.preventDefault();
									handleNotificationClick(n);
								}
							}}
							onclick={() => handleNotificationClick(n)}
							class="group flex items-start gap-3 p-3.5 transition-colors cursor-pointer {n.isRead
								? 'hover:bg-slate-elevated/60'
								: 'bg-aura-lavender/5 hover:bg-aura-lavender/10'}"
						>
							<!-- Avatar & Action Icon -->
							<div class="relative shrink-0 mt-0.5">
								<img
									src={n.sender?.avatar ||
										`https://api.dicebear.com/7.x/identicon/svg?seed=${n.sender?.username || 'user'}`}
									alt={n.sender?.username || 'User'}
									class="h-9 w-9 rounded-full object-cover border border-slate-border/80"
								/>
								<div
									class="absolute -bottom-1 -right-1 rounded-full p-0.5 text-white shadow-sm ring-1 ring-deep-slate {n.type ===
									'like'
										? 'bg-rose-500'
										: n.type === 'comment'
											? 'bg-sky-500'
											: n.type === 'reply'
												? 'bg-aura-lavender text-midnight-ink'
												: 'bg-emerald-500'}"
								>
									{#if n.type === 'like'}
										<Heart class="h-2.5 w-2.5 fill-current" />
									{:else if n.type === 'comment'}
										<MessageSquare class="h-2.5 w-2.5 fill-current" />
									{:else if n.type === 'reply'}
										<Reply class="h-2.5 w-2.5" />
									{:else}
										<Video class="h-2.5 w-2.5" />
									{/if}
								</div>
							</div>

							<!-- Content -->
							<div class="flex-1 min-w-0">
								<p class="text-xs leading-snug text-soft-ivory">
									<span class="font-semibold text-soft-ivory hover:underline">
										@{n.sender?.username || 'Someone'}
									</span>
									{#if n.type === 'like'}
										<span class="text-muted-silver"> liked your video</span>
									{:else if n.type === 'comment'}
										<span class="text-muted-silver"> commented on your video</span>
									{:else if n.type === 'reply'}
										<span class="text-muted-silver"> replied to your comment</span>
									{:else if n.type === 'new_video'}
										<span class="text-muted-silver"> uploaded a new video</span>
									{:else}
										<span class="text-muted-silver"> interacted with you</span>
									{/if}
								</p>

								{#if n.comment?.text}
									<p class="mt-1 text-xs text-muted-silver/80 line-clamp-2 italic bg-midnight-ink/40 p-1.5 rounded-lg border border-slate-border/50">
										"{n.comment.text}"
									</p>
								{:else if n.video?.title}
									<p class="mt-0.5 text-xs text-muted-silver line-clamp-1">
										{n.video.title}
									</p>
								{/if}

								<div class="mt-1 flex items-center gap-2 text-[11px] text-muted-silver/70">
									<span>{formatTimeAgo(n.createdAt)}</span>
									{#if !n.isRead}
										<span class="h-1.5 w-1.5 rounded-full bg-aura-lavender"></span>
									{/if}
								</div>
							</div>

							<!-- Thumbnail & Dismiss -->
							<div class="flex items-center gap-1 shrink-0">
								{#if n.video?.thumbnailUrl}
									<img
										src={n.video.thumbnailUrl}
										alt={n.video.title || 'Video'}
										class="h-10 w-14 rounded-lg object-cover border border-slate-border/60"
									/>
								{/if}
								<button
									type="button"
									onclick={(e) => handleDeleteNotification(e, n.id)}
									class="opacity-0 group-hover:opacity-100 text-muted-silver hover:text-destructive hover:bg-destructive/10 rounded-full p-1 transition-all"
									title="Dismiss notification"
								>
									<X class="h-3.5 w-3.5" />
								</button>
							</div>
						</div>
					{/each}
				{/if}
			</div>

			<!-- Footer -->
			<div class="border-t border-slate-border/60 px-4 py-2.5 text-center bg-midnight-ink/40">
				<a
					href="/notifications"
					onclick={closeDropdown}
					class="text-xs font-semibold text-aura-lavender hover:underline inline-flex items-center gap-1"
				>
					View all notifications &rarr;
				</a>
			</div>
		</div>
	{/if}
</div>
