<script lang="ts">
	import { page } from '$app/state';
	import { fetchApi } from '#lib/api';
	import { userState } from '#lib/user.svelte';
	import VideoThumbnail from '#lib/components/VideoThumbnail.svelte';
	import { Image as ImageIcon, Loader2 } from 'lucide-svelte';
	import { createQuery, createMutation } from '@tanstack/svelte-query';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { Button } from '#lib/components/ui/button';

	let username = $derived(page.params.username);

	let currentPage = $state(1);

	let profileQuery = createQuery(() => ({
		queryKey: ['profile', username, currentPage],
		queryFn: async () => {
			const data = await fetchApi(`/users/profile/${username}?page=${currentPage}&limit=10`);
			return data;
		}
	}));

	let isOwnProfile = $derived(userState.user?.id === profileQuery.data?.user?.id);

	const subscribeMutation = createMutation({
		mutationFn: async (userId: string) => {
			return await fetchApi(`/interactions/subscribe/${userId}`, {
				method: 'POST'
			});
		},
		onSuccess: () => {
			profileQuery.refetch();
		}
	});

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
		<div class="bg-muted relative h-48 w-full md:h-72">
			{#if profile.user.banner}
				<img
					src={profile.user.banner}
					alt={`${profile.user.username}'s banner`}
					class="h-full w-full object-cover"
				/>
			{:else}
				<div class="flex h-full w-full flex-col items-center justify-center opacity-40">
					<ImageIcon size={64} class="text-muted-foreground mb-4" />
					<span class="text-muted-foreground font-medium">No banner provided</span>
				</div>
			{/if}
		</div>

		<div class="container mx-auto px-4 sm:px-6">
			<!-- PROFILE HEADER INFO -->
			<div
				class="bg-card relative z-10 -mt-12 mb-10 flex flex-col gap-6 rounded-3xl border border-border p-6 shadow-sm sm:-mt-20 md:flex-row md:items-end md:justify-between"
			>
				<div class="flex w-full flex-col gap-6 md:flex-row md:items-end">
					<div class="shrink-0">
						<img
							src={profileQuery.data.user.avatar ||
								`https://api.dicebear.com/7.x/identicon/svg?seed=${profileQuery.data.user.username}`}
							alt={profileQuery.data.user.username}
							class="bg-card h-24 w-24 rounded-full object-cover ring-4 ring-background sm:h-36 sm:w-36"
						/>
					</div>

					<div class="grow pb-2">
						<h1
							class="text-foreground font-display text-3xl font-extrabold tracking-tight sm:text-4xl"
						>
							{profileQuery.data.user.username}
						</h1>
						<p class="text-muted-foreground mt-2 flex items-center gap-2 font-medium">
							<span class="text-foreground font-bold"
								>{profileQuery.data.user.subscriberCount || 0}</span
							>
							subscribers
							<span>•</span>
							<span class="text-foreground font-bold">{profileQuery.data.videos?.length || 0}</span
							> videos
						</p>

						{#if profileQuery.data.user.bio}
							<p
								class="text-muted-foreground mt-4 max-w-2xl text-sm whitespace-pre-line sm:text-base"
							>
								{profileQuery.data.user.bio}
							</p>
						{/if}
					</div>

					<div class="shrink-0 pb-2">
						{#if isOwnProfile}
							<Button variant="secondary" href="/profile" class="w-full md:w-auto">
								Customize Channel
							</Button>
						{:else if userState.user}
							<Button
								variant={profileQuery.data.user.isSubscribed ? 'secondary' : 'default'}
								disabled={subscribeMutation.isPending}
								onclick={handleSubscribe}
								class="w-full md:w-auto"
							>
								{#if subscribeMutation.isPending}
									<Loader2 class="mr-2 -ml-1 h-4 w-4 animate-spin" />
								{/if}
								{profileQuery.data.user.isSubscribed ? 'Subscribed' : 'Subscribe'}
							</Button>
						{/if}
					</div>
				</div>
			</div>

			<!-- UPLOADS SECTION -->
			<h2 class="text-foreground font-display mb-6 text-2xl font-bold tracking-tight">Uploads</h2>

			<div
				class="grid grid-cols-1 gap-6 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5"
			>
				{#if profileQuery.data.videos && profileQuery.data.videos.length > 0}
					{#each profileQuery.data.videos as video}
						<a href={`/watch/${video.fileId || video.id}`} class="group flex flex-col space-y-3">
							<!-- Thumbnail -->
							<VideoThumbnail
								{video}
								class="aspect-video rounded-xl"
								imgClass="transition-transform duration-300 group-hover:scale-105"
							>
								<div
									class="absolute right-2 bottom-2 rounded bg-black/80 px-2 py-1 text-xs font-medium text-white backdrop-blur-sm"
								>
									{formatDuration(video.duration)}
								</div>
							</VideoThumbnail>

							<!-- Metadata -->
							<div class="flex gap-3">
								<div class="flex flex-col overflow-hidden">
									<h3
										class="line-clamp-2 text-sm leading-tight font-semibold transition-colors group-hover:text-primary"
									>
										{video.title}
									</h3>
									<div class="text-muted-foreground mt-1 space-y-0.5 text-xs">
										<div class="flex items-center gap-1">
											<span>{video.views || 0} views</span>
											<span class="text-[10px]">•</span>
											<span>{new Date(video.created_at).toLocaleDateString()}</span>
										</div>
									</div>
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
