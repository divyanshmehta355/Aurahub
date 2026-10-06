<script lang="ts">
	import { fetchApi } from '#lib/api';
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

	let activeFilter = $state<'all' | 'unread'>('all');

	const notificationsQuery = createQuery(
		() => ({
			queryKey: ['notifications'],
			queryFn: async () => {
				if (!userState.user && userState.isLoaded) {
					window.location.href = '/login';
					return { notifications: [], unreadCount: 0 };
				}
				const res = await fetchApi('/notifications?limit=50').catch(() => ({
					notifications: [],
					unreadCount: 0
				}));
				return res || { notifications: [], unreadCount: 0 };
			},
			enabled: !!userState.user
		}),
		() => queryClient
	);

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

	function handleNotificationClick(n: any) {
		if (!n.isRead) {
			markSingleReadMutation.mutate(n.id);
		}
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
		if (!confirm('Are you sure you want to clear all notifications?')) return;
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
		const hours = Math.floor(seconds / 60);
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

<svelte:head>
	<title>Notifications - Aurahub</title>
</svelte:head>

<div class="mx-auto max-w-4xl space-y-6 px-4 py-4 md:px-6 md:py-8">
	<!-- Page Header -->
	<div class="flex flex-col justify-between gap-4 sm:flex-row sm:items-center border-b pb-4">
		<div>
			<h1 class="text-2xl font-bold tracking-tight md:text-3xl flex items-center gap-2.5">
				Notifications
				{#if unreadCount > 0}
					<span
						class="rounded-full bg-primary/10 px-2.5 py-0.5 text-xs font-semibold text-primary"
					>
						{unreadCount} unread
					</span>
				{/if}
			</h1>
			<p class="text-muted-foreground mt-1 text-sm md:text-base">
				Activity on your channel, videos, and comments.
			</p>
		</div>

		<div class="flex items-center gap-2">
			{#if unreadCount > 0}
				<Button
					variant="outline"
					size="sm"
					onclick={handleMarkAllRead}
					disabled={markAllReadMutation.isPending}
				>
					<CheckCheck class="mr-2 h-4 w-4" />
					Mark all as read
				</Button>
			{/if}
			{#if notifications.length > 0}
				<Button
					variant="ghost"
					size="sm"
					onclick={handleClearAll}
					disabled={clearAllMutation.isPending}
					class="text-muted-foreground hover:text-destructive"
				>
					<Trash2 class="mr-2 h-4 w-4" />
					Clear all
				</Button>
			{/if}
		</div>
	</div>

	<!-- Filter Tabs -->
	<div class="flex border-b border-border/80 text-sm font-medium">
		<button
			type="button"
			onclick={() => (activeFilter = 'all')}
			class="py-2.5 px-4 border-b-2 transition-colors {activeFilter === 'all'
				? 'border-primary text-primary font-semibold'
				: 'border-transparent text-muted-foreground hover:text-foreground'}"
		>
			All ({notifications.length})
		</button>
		<button
			type="button"
			onclick={() => (activeFilter = 'unread')}
			class="py-2.5 px-4 border-b-2 transition-colors {activeFilter === 'unread'
				? 'border-primary text-primary font-semibold'
				: 'border-transparent text-muted-foreground hover:text-foreground'}"
		>
			Unread ({unreadCount})
		</button>
	</div>

	<!-- Notifications List Area -->
	<div class="min-h-[400px]">
		{#if notificationsQuery.isPending}
			<div class="flex flex-col items-center justify-center py-16 text-muted-foreground gap-3">
				<Loader2 class="h-8 w-8 animate-spin text-primary" />
				<span class="text-sm font-medium">Loading notifications...</span>
			</div>
		{:else if filteredNotifications.length === 0}
			<div
				class="flex flex-col items-center justify-center py-20 px-4 text-center rounded-2xl border bg-card shadow-sm"
			>
				<div class="rounded-full bg-muted/60 p-5 mb-4 text-muted-foreground">
					<Sparkles class="h-8 w-8 text-primary/70" />
				</div>
				<h3 class="font-semibold text-lg">No notifications here</h3>
				<p class="text-muted-foreground text-sm mt-1 max-w-sm">
					{activeFilter === 'unread'
						? "You've read everything! There are no unread notifications."
						: 'When viewers like, comment, or interact with your content, you will see notifications here.'}
				</p>
			</div>
		{:else}
			<div class="rounded-2xl border bg-card shadow-sm overflow-hidden divide-y divide-border/50">
				{#each filteredNotifications as n (n.id)}
					<div
						role="button"
						tabindex="0"
						onclick={() => handleNotificationClick(n)}
						onkeydown={(e) => e.key === 'Enter' && handleNotificationClick(n)}
						class="group flex items-start gap-4 p-4 md:p-5 transition-colors cursor-pointer text-left {n.isRead
							? 'hover:bg-muted/40'
							: 'bg-primary/5 hover:bg-primary/10'}"
					>
						<!-- Sender Avatar with Type Badge -->
						<div class="relative shrink-0 mt-0.5">
							<img
								src={n.sender?.avatar ||
									`https://api.dicebear.com/7.x/identicon/svg?seed=${n.sender?.username || 'user'}`}
								alt={n.sender?.username || 'User'}
								class="h-10 w-10 md:h-12 md:w-12 rounded-full object-cover border border-border"
							/>
							<div
								class="absolute -bottom-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full text-white shadow-sm ring-2 ring-background {n.type ===
								'like'
									? 'bg-rose-500'
									: n.type === 'comment'
										? 'bg-sky-500'
										: n.type === 'reply'
											? 'bg-purple-500'
											: 'bg-emerald-500'}"
							>
								{#if n.type === 'like'}
									<Heart class="h-3 w-3 fill-current" />
								{:else if n.type === 'comment'}
									<MessageSquare class="h-3 w-3 fill-current" />
								{:else if n.type === 'reply'}
									<Reply class="h-3 w-3" />
								{:else}
									<Video class="h-3 w-3" />
								{/if}
							</div>
						</div>

						<!-- Content Details -->
						<div class="flex-1 min-w-0 pr-2">
							<p class="text-sm md:text-base leading-relaxed text-foreground">
								<span class="font-semibold text-foreground hover:underline"
									>@{n.sender?.username || 'Someone'}</span
								>
								{#if n.type === 'like'}
									<span> liked your video </span>
									{#if n.video?.title}
										<span class="font-medium text-foreground">"{n.video.title}"</span>
									{/if}
								{:else if n.type === 'comment'}
									<span> commented on </span>
									{#if n.video?.title}
										<span class="font-medium text-foreground">"{n.video.title}"</span>
									{/if}
									{#if n.comment?.text}
										<span class="block mt-1 italic text-muted-foreground text-sm line-clamp-2"
											>"{n.comment.text}"</span
										>
									{/if}
								{:else if n.type === 'reply'}
									<span> replied to your comment on </span>
									{#if n.video?.title}
										<span class="font-medium text-foreground">"{n.video.title}"</span>
									{/if}
									{#if n.comment?.text}
										<span class="block mt-1 italic text-muted-foreground text-sm line-clamp-2"
											>"{n.comment.text}"</span
										>
									{/if}
								{:else if n.type === 'new_video'}
									<span> uploaded a new video: </span>
									{#if n.video?.title}
										<span class="font-medium text-foreground">"{n.video.title}"</span>
									{/if}
								{/if}
							</p>
							<div class="flex items-center gap-3 mt-1.5">
								<span class="text-xs text-muted-foreground font-medium">
									{formatTimeAgo(n.createdAt)}
								</span>
								{#if !n.isRead}
									<span class="inline-flex items-center gap-1 text-[11px] font-semibold text-primary">
										<span class="h-2 w-2 rounded-full bg-primary"></span>
										New
									</span>
								{/if}
							</div>
						</div>

						<!-- Video Thumbnail or Action -->
						<div class="flex items-center gap-2 shrink-0">
							{#if n.video?.thumbnailUrl}
								<img
									src={n.video.thumbnailUrl}
									alt={n.video.title || 'Video'}
									class="h-12 w-20 md:h-14 md:w-24 rounded-lg object-cover border border-border"
								/>
							{/if}
							<button
								type="button"
								onclick={(e) => handleDeleteNotification(e, n.id)}
								class="opacity-0 group-hover:opacity-100 text-muted-foreground hover:text-destructive hover:bg-destructive/10 rounded-lg p-2 transition-all"
								title="Dismiss notification"
							>
								<X class="h-4 w-4" />
							</button>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</div>
</div>
