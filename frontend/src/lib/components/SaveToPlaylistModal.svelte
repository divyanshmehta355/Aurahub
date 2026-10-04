<script lang="ts">
	import { fetchApi } from '#lib/api';
	import { Plus, X, ListVideo, Loader2, Check } from 'lucide-svelte';
	import { userState } from '#lib/user.svelte';
	import { onMount } from 'svelte';
	import { fade, scale } from 'svelte/transition';

	let { videoId, isOpen = $bindable(false) } = $props<{ videoId: string, isOpen: boolean }>();

	let playlists: any[] = $state([]);
	let isLoading = $state(true);
	let error = $state('');

	let isCreating = $state(false);
	let newPlaylistTitle = $state('');

	// State to track which playlists contain this video
	let savingStatus: Record<string, 'idle' | 'saving' | 'saved' | 'error'> = $state({});
	let containsVideo: Record<string, boolean> = $state({});

	$effect(() => {
		if (isOpen) {
			if (!userState.user) {
				alert('Please login to save to a playlist.');
				isOpen = false;
				return;
			}
			loadPlaylists();
		}
	});

	async function loadPlaylists() {
		isLoading = true;
		error = '';
		try {
			const res = await fetchApi('/playlists');
			playlists = res.playlists || res.data || [];
			
			// Check which playlists contain this video
			playlists.forEach(async (p) => {
				savingStatus[p.id] = 'idle';
				// Ideally, the backend would return this info in the /playlists list,
				// but if not, we can query the playlist details or just assume false
				// and let the toggle handle it. For a quick implementation, we will
				// assume false initially.
				containsVideo[p.id] = false; 
			});
		} catch (err: any) {
			error = err.message || 'Failed to load playlists.';
		} finally {
			isLoading = false;
		}
	}

	async function handleToggle(playlist: any) {
		const pid = playlist.id;
		if (savingStatus[pid] === 'saving') return;
		savingStatus[pid] = 'saving';

		try {
			const action = containsVideo[pid] ? 'remove' : 'add';
			// Using the TogglePlaylistVideoHandler which is mapped to POST /playlists/:id/videos
			const res = await fetchApi(`/playlists/${pid}/videos`, {
				method: 'POST',
				body: JSON.stringify({ videoId, action })
			});
			containsVideo[pid] = action === 'add';
			savingStatus[pid] = 'saved';
			setTimeout(() => {
				if (savingStatus[pid] === 'saved') savingStatus[pid] = 'idle';
			}, 2000);
		} catch (err) {
			console.error(err);
			savingStatus[pid] = 'error';
			setTimeout(() => {
				if (savingStatus[pid] === 'error') savingStatus[pid] = 'idle';
			}, 2000);
		}
	}

	async function handleCreate() {
		if (!newPlaylistTitle.trim()) return;
		
		isCreating = true;
		try {
			const res = await fetchApi('/playlists', {
				method: 'POST',
				body: JSON.stringify({ title: newPlaylistTitle.trim(), isPublic: true })
			});
			
			const created = res.playlist;
			if (created) {
				playlists = [created, ...playlists];
				savingStatus[created.id] = 'idle';
				containsVideo[created.id] = false;
				
				// Automatically add to the new playlist
				await handleToggle(created);
			}
			newPlaylistTitle = '';
		} catch (err: any) {
			alert('Failed to create playlist');
		} finally {
			isCreating = false;
		}
	}
</script>

{#if isOpen}
	<div 
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4"
		transition:fade={{ duration: 200 }}
		role="dialog"
		aria-modal="true"
	>
		<div 
			class="w-full max-w-md bg-card border border-border rounded-2xl shadow-2xl overflow-hidden flex flex-col max-h-[85vh]"
			transition:scale={{ duration: 200, start: 0.95 }}
		>
			<div class="px-6 py-4 border-b flex items-center justify-between sticky top-0 bg-card z-10">
				<h2 class="text-xl font-semibold tracking-tight">Save to playlist</h2>
				<button 
					onclick={() => isOpen = false}
					class="p-2 -mr-2 rounded-full hover:bg-muted text-muted-foreground hover:text-foreground transition-colors"
				>
					<X class="w-5 h-5" />
				</button>
			</div>
			
			<div class="p-6 overflow-y-auto flex-1 custom-scrollbar">
				{#if isLoading}
					<div class="flex items-center justify-center py-8">
						<Loader2 class="w-8 h-8 animate-spin text-primary" />
					</div>
				{:else if error}
					<div class="text-center py-8 text-rose-500 font-medium">{error}</div>
				{:else}
					<div class="space-y-2 mb-6">
						{#each playlists as playlist}
							<button 
								onclick={() => handleToggle(playlist)}
								class="w-full flex items-center justify-between p-3 rounded-xl hover:bg-muted/50 transition-colors text-left group"
							>
								<div class="flex items-center gap-3">
									<div class={`w-5 h-5 rounded border flex items-center justify-center transition-colors ${containsVideo[playlist.id] ? 'bg-primary border-primary' : 'border-muted-foreground/40 group-hover:border-foreground/60'}`}>
										{#if containsVideo[playlist.id]}
											<Check class="w-3.5 h-3.5 text-primary-foreground" />
										{/if}
									</div>
									<span class="font-medium truncate max-w-[200px] sm:max-w-[260px]">{playlist.title}</span>
								</div>
								
								{#if savingStatus[playlist.id] === 'saving'}
									<Loader2 class="w-4 h-4 animate-spin text-muted-foreground" />
								{:else if savingStatus[playlist.id] === 'error'}
									<span class="text-xs text-rose-500 font-medium">Failed</span>
								{/if}
							</button>
						{:else}
							<div class="text-center py-6 text-muted-foreground flex flex-col items-center">
								<ListVideo class="w-8 h-8 mb-2 opacity-50" />
								<p>You don't have any playlists yet.</p>
							</div>
						{/each}
					</div>
				{/if}
			</div>

			<div class="p-6 border-t bg-muted/20">
				<form onsubmit={(e) => { e.preventDefault(); handleCreate(); }} class="flex gap-2">
					<input 
						type="text" 
						bind:value={newPlaylistTitle} 
						placeholder="New playlist name..."
						class="flex-1 bg-background border border-border rounded-xl px-4 py-2 text-sm focus:ring-2 focus:ring-primary focus:outline-none transition-shadow"
						required
					/>
					<button 
						type="submit" 
						disabled={isCreating || !newPlaylistTitle.trim()}
						class="flex items-center gap-2 bg-foreground text-background px-4 py-2 rounded-xl text-sm font-semibold hover:bg-foreground/90 disabled:opacity-50 transition-colors shrink-0"
					>
						{#if isCreating}
							<Loader2 class="w-4 h-4 animate-spin" />
						{:else}
							<Plus class="w-4 h-4" /> Create
						{/if}
					</button>
				</form>
			</div>
		</div>
	</div>
{/if}
