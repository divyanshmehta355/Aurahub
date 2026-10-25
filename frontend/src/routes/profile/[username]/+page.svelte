<script lang="ts">
	import { page } from '$app/state';
	import { fetchApi } from '#lib/api';
	import { userState } from '#lib/user.svelte';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import { Image as ImageIcon, Loader2 } from 'lucide-svelte';
	import { createQuery, createMutation } from '@tanstack/svelte-query';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { Button } from '#lib/components/ui/button';
	import { contentModeState } from '#lib/consent.svelte';

	let username = $derived(page.params.username);

	let currentPage = $state(1);

	let profileQuery = createQuery(() => ({
		queryKey: ['profile', username, currentPage, contentModeState.isAdult],
		queryFn: async () => {
			const data = await fetchApi(`/users/profile/${username}?page=${currentPage}&limit=10`);
			return data;
		}
	}));

	let isOwnProfile = $derived(userState.user?.id === profileQuery.data?.user?.id);

	const subscribeMutation = createMutation(() => ({
		mutationFn: async (userId: string) => {
			return await fetchApi(`/interactions/subscribe/${userId}`, {
				method: 'POST'
			});
		},
		onSuccess: () => {
			profileQuery.refetch();
		}
	}));

	async function changePage(newPage: number) {
		if (newPage < 1 || (!profileQuery.data?.hasMore && newPage > currentPage)) return;
		currentPage = newPage;
		window.scrollTo({ top: 0, behavior: 'smooth' });
	}

	async function handleSubscribe() {
		if (!userState.user) {
			alert('Please log in to subscribe.');
			return;
		}

		if (profileQuery.data?.user?.id) {
			subscribeMutation.mutate(profileQuery.data.user.id);
		}
	}

	function formatDuration(seconds: number) {
		if (!seconds) return '0:00';
		const m = Math.floor(seconds / 60);
		const s = Math.floor(seconds % 60);
		return `${m}:${s.toString().padStart(2, '0')}`;
	}
</script>

<svelte:head>
	<title>{username ? `${username} - Aurahub` : 'Profile - Aurahub'}</title>
</svelte:head>

{#if profileQuery.isPending}
	<main class="pb-12">
		<Skeleton class="mb-8 h-48 w-full rounded-none md:h-64" />
		<div class="container mx-auto px-6">
			<div class="-mt-20 mb-8 flex items-end gap-6">
				<Skeleton class="h-32 w-32 rounded-full ring-4 ring-background" />
				<Skeleton class="mb-2 h-10 w-48 rounded" />
			</div>
			<div class="grid grid-cols-1 gap-6 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
				{#each Array(4) as _}
					<div class="flex flex-col space-y-3">
						<Skeleton class="aspect-video w-full rounded-xl" />
						<div class="flex gap-3">
							<Skeleton class="h-10 w-10 shrink-0 rounded-full" />
							<div class="w-full space-y-2">
								<Skeleton class="h-4 w-3/4 rounded" />
								<Skeleton class="h-3 w-1/2 rounded" />
							</div>
						</div>
					</div>
				{/each}
			</div>
		</div>
	</main>
{:else if profileQuery.isError || !profileQuery.data}
	<div class="flex flex-col items-center justify-center py-12 text-center">
		<div class="bg-destructive/10 text-destructive rounded-xl p-4">
			<p>{profileQuery.error?.message || 'User not found'}</p>
			<Button variant="outline" class="mt-4" onclick={() => profileQuery.refetch()}
				>Try again</Button
			>
		</div>
	</div>
{:else}
	<main class="fade-in pb-12">
		<!-- CHANNEL BANNER -->
		<div class="bg-slate-surface border-b border-border/60 relative h-52 w-full md:h-80 overflow-hidden">
			{#if profileQuery.data.user.banner}
				<img
					src={profileQuery.data.user.banner}
					alt={`${profileQuery.data.user.username}'s banner`}
					class="h-full w-full object-cover"
				/>
			{:else}
				<div class="flex h-full w-full flex-col items-center justify-center opacity-40 bg-gradient-to-br from-deep-slate to-midnight-ink">
					<ImageIcon size={64} class="text-muted-silver mb-3" />
					<span class="text-muted-silver text-xs font-semibold uppercase tracking-wider">No banner configured</span>
				</div>
			{/if}
			<div class="absolute inset-0 bg-gradient-to-t from-background/90 via-transparent to-transparent"></div>
		</div>

		<div class="container mx-auto px-4 sm:px-6">
			<!-- PROFILE HEADER INFO -->
			<div
				class="bg-card relative z-10 -mt-16 mb-10 flex flex-col gap-6 rounded-3xl border border-border/70 p-6 sm:p-8 shadow-2xl backdrop-blur-xl sm:-mt-20 md:flex-row md:items-end md:justify-between"
			>
				<div class="flex w-full flex-col gap-6 md:flex-row md:items-end">
					<div class="shrink-0">
						<img
							src={profileQuery.data.user.avatar ||
								`https://api.dicebear.com/7.x/identicon/svg?seed=${profileQuery.data.user.username}`}
							alt={profileQuery.data.user.username}
							class="h-24 w-24 rounded-full object-cover ring-4 ring-background border border-border/80 sm:h-36 sm:w-36 shadow-lg"
						/>
					</div>

					<div class="grow pb-1">
						<h1
							class="text-foreground text-3xl font-extrabold tracking-tight sm:text-4xl"
						>
							{profileQuery.data.user.username}
						</h1>
						<p class="text-muted-silver mt-2 flex items-center gap-2 text-xs sm:text-sm font-medium">
							<span class="text-foreground font-bold"
								>{profileQuery.data.user.subscriberCount || 0}</span
							>
							subscribers
							<span>•</span>
							<span class="text-foreground font-bold">{profileQuery.data.videos?.length || 0}</span> releases
						</p>

						{#if profileQuery.data.user.bio}
							<p
								class="text-muted-foreground mt-3 max-w-2xl text-xs sm:text-sm whitespace-pre-line leading-relaxed"
							>
								{profileQuery.data.user.bio}
							</p>
						{/if}
					</div>

					<div class="shrink-0 pb-1">
						{#if isOwnProfile}
							<Button variant="secondary" href="/profile" class="w-full md:w-auto rounded-full px-5 font-semibold text-xs border border-border/70">
								Customize Channel
							</Button>
						{:else if userState.user}
							<Button
								variant={profileQuery.data.user.isSubscribed ? 'secondary' : 'default'}
								disabled={subscribeMutation.isPending}
								onclick={handleSubscribe}
								class="w-full md:w-auto rounded-full px-6 font-semibold text-xs shadow-xs"
							>
								{#if subscribeMutation.isPending}
									<Loader2 class="mr-2 -ml-1 h-3.5 w-3.5 animate-spin" />
								{/if}
								{profileQuery.data.user.isSubscribed ? 'Subscribed' : 'Subscribe'}
							</Button>
						{/if}
					</div>
				</div>
			</div>

			<!-- UPLOADS SECTION -->
			<div class="flex items-center justify-between border-b border-border/50 pb-3 mb-6">
				<h2 class="text-foreground text-xl font-bold tracking-tight">Releases</h2>
				<span class="text-xs text-muted-silver">{profileQuery.data.videos?.length || 0} videos</span>
			</div>

			<div
				class="grid grid-cols-1 gap-6 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5"
			>
				{#if profileQuery.data.videos && profileQuery.data.videos.length > 0}
					{#each profileQuery.data.videos.filter((v: any) => !v.isShort) as video}
						<a href={`/watch/${video.fileId || video.id}`} class="group flex flex-col space-y-3 min-w-0 transition-transform">
							<!-- Thumbnail -->
							<div class="relative overflow-hidden rounded-2xl border border-border/60 bg-card shadow-xs group-hover:border-primary/40 group-hover:shadow-md transition-all">
								<VideoThumbnail
									{video}
									class="aspect-video w-full rounded-2xl"
									imgClass="transition-transform duration-500 ease-out group-hover:scale-105"
								>
									{#if video.duration}
										<div
											class="absolute right-2 bottom-2 rounded-md bg-black/80 px-2 py-0.5 text-[11px] font-semibold text-white backdrop-blur-md border border-white/10"
										>
											{formatDuration(video.duration)}
										</div>
									{/if}
								</VideoThumbnail>
							</div>

							<!-- Metadata -->
							<div class="flex flex-col overflow-hidden min-w-0 px-0.5">
								<h3
									class="line-clamp-2 text-xs sm:text-sm leading-snug font-semibold text-foreground transition-colors group-hover:text-primary break-words [overflow-wrap:anywhere]"
									title={video.title}
								>
									{video.title}
								</h3>
								<div class="text-muted-silver mt-1 text-[11px] flex items-center gap-1.5">
									<span>{video.views || 0} views</span>
									<span>•</span>
									<span>{new Date(video.created_at || video.createdAt || Date.now()).toLocaleDateString()}</span>
								</div>
							</div>
						</a>
					{/each}
				{:else}
					<div
						class="bg-card col-span-full rounded-2xl border border-border p-16 text-center shadow-sm"
					>
						<p class="text-muted-foreground text-lg font-medium">
							This creator hasn't uploaded any videos yet.
						</p>
						<p class="text-muted-foreground mt-2 text-sm">Check back later for new content!</p>
					</div>
				{/if}
				<!-- Pagination Controls -->
				<div class="mt-12 flex items-center justify-between border-t border-border pt-6 pb-8">
					<Button
						variant="secondary"
						onclick={() => changePage(currentPage - 1)}
						disabled={currentPage === 1 || profileQuery.isFetching}
						class="rounded-full px-6"
					>
						Previous
					</Button>
					<span class="text-muted-foreground text-sm font-medium">
						Page {currentPage}
					</span>
					<Button
						variant="default"
						onclick={() => changePage(currentPage + 1)}
						disabled={!profileQuery.data.hasMore || profileQuery.isFetching}
						class="rounded-full px-6"
					>
						Next
					</Button>
				</div>
			</div>
		</div>
	</main>
{/if}

<style>
	.fade-in {
		animation: fadeIn 0.4s ease-in-out;
	}

	@keyframes fadeIn {
		from {
			opacity: 0;
			transform: translateY(20px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}
</style>
