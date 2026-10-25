<script lang="ts">
	import { MessageSquare, Loader2, Trash2 } from 'lucide-svelte';
	import { Button } from '#lib/components/ui/button';
	import { Input } from '#lib/components/ui/input';
	import { Skeleton } from '#lib/components/ui/skeleton';
	import { fetchApi } from '#lib/api';
	import { userState } from '#lib/user.svelte';
	import { createQuery, createMutation, useQueryClient } from '@tanstack/svelte-query';

	let {
		videoId = '',
		resolvedVideoId = '',
		uploaderUsername = '',
		uploaderId = ''
	}: {
		videoId?: string;
		resolvedVideoId?: string;
		uploaderUsername?: string;
		uploaderId?: string;
	} = $props();

	const queryClient = useQueryClient();
	let newCommentText = $state('');

	let activeVideoId = $derived(resolvedVideoId || videoId);

	const commentsQuery = createQuery(
		() => ({
			queryKey: ['comments', activeVideoId],
			queryFn: async () => {
				const res = await fetchApi(`/comments/${activeVideoId}`);
				return res.comments || res.data || [];
			}
		}),
		() => queryClient
	);

	const commentMutation = createMutation(
		() => ({
			mutationFn: async (content: string) => {
				return await fetchApi('/comments', {
					method: 'POST',
					body: JSON.stringify({ videoId: activeVideoId, content, text: content })
				});
			},
			onSuccess: () => {
				newCommentText = '';
				queryClient.invalidateQueries({ queryKey: ['comments', activeVideoId] });
				if (videoId !== activeVideoId) {
					queryClient.invalidateQueries({ queryKey: ['comments', videoId] });
				}
			}
		}),
		() => queryClient
	);

	const deleteCommentMutation = createMutation(
		() => ({
			mutationFn: async (commentId: string) => {
				return await fetchApi(`/comments/${commentId}`, {
					method: 'DELETE'
				});
			},
			onSuccess: () => {
				queryClient.invalidateQueries({ queryKey: ['comments', activeVideoId] });
				if (videoId !== activeVideoId) {
					queryClient.invalidateQueries({ queryKey: ['comments', videoId] });
				}
			}
		}),
		() => queryClient
	);

	function handleDeleteComment(commentId: string) {
		if (!confirm('Are you sure you want to delete this comment?')) return;
		deleteCommentMutation.mutate(commentId);
	}

	function handleCommentSubmit() {
		if (!newCommentText.trim() || !userState.user) return;
		commentMutation.mutate(newCommentText.trim());
	}

	function canDeleteComment(comment: any): boolean {
		if (!userState.user) return false;
		const uid = userState.user.id;
		const uname = userState.user.username;
		const authorId =
			comment.authorId ||
			comment.author?.id ||
			comment.user?.id ||
			comment.author?._id ||
			comment.user?._id;
		const authorName = comment.author?.username || comment.user?.username;
		return (
			uid === authorId ||
			uname === authorName ||
			uid === uploaderId ||
			(!!uploaderUsername && uname === uploaderUsername)
		);
	}

	function formatTimeAgo(dateString: string) {
		const date = new Date(dateString);
		const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000);
		let interval = seconds / 31536000;
		if (interval > 1) return Math.floor(interval) + ' years ago';
		interval = seconds / 2592000;
		if (interval > 1) return Math.floor(interval) + ' months ago';
		interval = seconds / 86400;
		if (interval > 1) return Math.floor(interval) + ' days ago';
		interval = seconds / 3600;
		if (interval > 1) return Math.floor(interval) + ' hours ago';
		interval = seconds / 60;
		if (interval > 1) return Math.floor(interval) + ' minutes ago';
		return Math.floor(seconds) + ' seconds ago';
	}
</script>

<div class="mt-8 border-t border-border/50 pt-6">
	<h3 class="mb-6 flex items-center gap-2.5 text-lg font-bold tracking-tight text-foreground">
		<MessageSquare class="h-5 w-5 text-primary" />
		<span>{commentsQuery.data?.length || 0} Comments</span>
	</h3>

	<!-- Post Comment Box -->
	<div class="mb-8 flex gap-3.5">
		<img
			src={userState.user?.avatar || `https://api.dicebear.com/7.x/identicon/svg?seed=Guest`}
			alt="Your Avatar"
			class="h-10 w-10 shrink-0 rounded-full border border-border/80 object-cover mt-0.5"
		/>
		<div class="flex-1 space-y-2">
			<Input
				type="text"
				bind:value={newCommentText}
				placeholder={userState.user ? 'Share your thoughts on this release...' : 'Sign in to join the conversation...'}
				disabled={!userState.user || commentMutation.isPending}
				class="bg-card border-border/70 rounded-2xl px-4 py-3 text-sm text-foreground focus-visible:border-primary focus-visible:ring-2 focus-visible:ring-primary/20 transition-all placeholder:text-muted-foreground/70"
			/>
			{#if newCommentText.length > 0}
				<div class="mt-2 flex justify-end gap-2">
					<Button variant="ghost" size="sm" class="rounded-full text-xs" onclick={() => (newCommentText = '')}>Cancel</Button>
					<Button size="sm" class="rounded-full px-5 text-xs font-semibold" onclick={handleCommentSubmit} disabled={commentMutation.isPending}>
						{#if commentMutation.isPending}
							<Loader2 class="mr-2 h-3.5 w-3.5 animate-spin" />
						{/if}
						Comment
					</Button>
				</div>
			{/if}
		</div>
	</div>

	<!-- Comments List -->
	<div class="space-y-5">
		{#if commentsQuery.isPending}
			{#each Array(3) as _}
				<div class="flex animate-pulse gap-3.5">
					<Skeleton class="h-10 w-10 rounded-full shrink-0" />
					<div class="flex-1 space-y-2 py-1">
						<Skeleton class="h-4 w-1/4 rounded-lg" />
						<Skeleton class="h-4 w-3/4 rounded-lg" />
					</div>
				</div>
			{/each}
		{:else if commentsQuery.data}
			{#each commentsQuery.data as comment}
				{@const isUploader = (comment.user?.username || comment.author?.username) === uploaderUsername}
				<div class="group flex items-start gap-3.5 rounded-2xl p-2.5 transition-colors hover:bg-secondary/40">
					<img
						src={comment.user?.avatar ||
							comment.author?.avatar ||
							`https://api.dicebear.com/7.x/identicon/svg?seed=${comment.user?.username || comment.author?.username || 'user'}`}
						alt="Avatar"
						class="h-10 w-10 shrink-0 rounded-full border border-border/80 object-cover mt-0.5"
					/>
					<div class="flex-1 space-y-1 min-w-0">
						<div class="flex items-baseline justify-between gap-2">
							<div class="flex items-center gap-2 flex-wrap">
								<span class="text-xs sm:text-sm font-semibold text-foreground">
									@{comment.user?.username || comment.author?.username || 'User'}
								</span>
								{#if isUploader}
									<span class="rounded-full bg-primary/20 border border-primary/30 px-2 py-0.5 text-[10px] font-bold text-primary">
										Creator
									</span>
								{/if}
								<span class="text-muted-silver text-[11px]">
									{formatTimeAgo(comment.createdAt || comment.created_at)}
								</span>
							</div>
							{#if canDeleteComment(comment)}
								<button
									type="button"
									onclick={() => handleDeleteComment(comment.id || comment._id)}
									disabled={deleteCommentMutation.isPending}
									class="text-muted-foreground hover:text-rose-400 rounded-lg p-1 opacity-0 transition-opacity group-hover:opacity-100"
									title="Delete comment"
								>
									<Trash2 class="h-3.5 w-3.5" />
								</button>
							{/if}
						</div>
						<p class="text-xs sm:text-sm leading-relaxed text-zinc-300 whitespace-pre-line break-words [overflow-wrap:anywhere]">
							{comment.content || comment.text}
						</p>
					</div>
				</div>
			{/each}
		{/if}
	</div>
</div>
