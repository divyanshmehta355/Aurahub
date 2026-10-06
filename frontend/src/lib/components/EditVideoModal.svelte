<script lang="ts">
	import { Dialog, DialogContent, DialogHeader, DialogTitle } from '#lib/components/ui/dialog';
	import { Input } from '#lib/components/ui/input';
	import { Label } from '#lib/components/ui/label';
	import { Textarea } from '#lib/components/ui/textarea';
	import { Button } from '#lib/components/ui/button';
	import { fetchApi } from '#lib/api';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import { Loader2, Image as ImageIcon } from 'lucide-svelte';

	let {
		video,
		open = $bindable(false),
		onClose
	} = $props<{ video: any; open: boolean; onClose?: () => void }>();

	const queryClient = useQueryClient();

	let title = $state('');
	let description = $state('');
	let category = $state('Other');
	let tags = $state('');
	let visibility = $state('public');
	let isAdult = $state(false);

	let thumbnailFile: File | null = $state(null);
	let error = $state('');

	// Sync local state when video prop changes or modal opens
	$effect(() => {
		if (open && video) {
			title = video.title || '';
			description = video.description || '';
			category = video.category || 'Other';
			tags = Array.isArray(video.tags) ? video.tags.join(', ') : '';
			visibility = video.visibility || 'public';
			isAdult = !!video.isAdult;
			thumbnailFile = null;
			error = '';
		}
	});

	const updateMutation = createMutation(
		() => ({
			mutationFn: async () => {
				// Update metadata
				const metaReq = fetchApi(`/videos/${video.id}`, {
					method: 'PUT',
					body: JSON.stringify({
						title,
						description,
						category,
						visibility,
						isAdult,
						tags: tags
							.split(',')
							.map((t) => t.trim())
							.filter(Boolean)
					})
				});

				// Update thumbnail if provided
				if (thumbnailFile) {
					const formData = new FormData();
					formData.append('thumbnailFile', thumbnailFile);
					const thumbReq = fetchApi(`/videos/${video.id}/update-thumbnail`, {
						method: 'POST',
						body: formData
					});
					await Promise.all([metaReq, thumbReq]);
				} else {
					await metaReq;
				}
			},
			onSuccess: () => {
				queryClient.invalidateQueries({ queryKey: ['dashboard'] });
				open = false;
				if (onClose) onClose();
			},
			onError: (err: any) => {
				error = err.message || 'Failed to update video';
			}
		}),
		() => queryClient
	);

	function handleSubmit(e: Event) {
		e.preventDefault();
		if (!title.trim()) {
			error = 'Title is required';
			return;
		}
		updateMutation.mutate();
	}
</script>

<Dialog bind:open>
	<DialogContent class="max-h-[90vh] overflow-y-auto sm:max-w-[600px]">
		<DialogHeader>
			<DialogTitle>Edit Video Details</DialogTitle>
		</DialogHeader>

		{#if error}
			<div class="bg-destructive/10 text-destructive rounded-lg p-3 text-sm font-medium">
				{error}
			</div>
		{/if}

		<form onsubmit={handleSubmit} class="space-y-5 py-4">
			<div class="space-y-1.5">
				<Label for="title">Title <span class="text-destructive">*</span></Label>
				<Input id="title" bind:value={title} required disabled={updateMutation.isPending} />
			</div>

			<div class="space-y-1.5">
				<Label for="description">Description</Label>
				<Textarea
					id="description"
					bind:value={description}
					rows={3}
					disabled={updateMutation.isPending}
				/>
			</div>

			<div class="grid grid-cols-2 gap-4">
				<div class="space-y-1.5">
					<Label for="category">Category</Label>
					<select
						id="category"
						bind:value={category}
						disabled={updateMutation.isPending}
						class="border-input h-10 w-full rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-primary focus:outline-none disabled:opacity-50"
					>
						<option value="Other">Other</option>
						<option value="Gaming">Gaming</option>
						<option value="Education">Education</option>
						<option value="Entertainment">Entertainment</option>
						<option value="Music">Music</option>
						<option value="Science">Science & Tech</option>
					</select>
				</div>
				<div class="space-y-1.5">
					<Label for="visibility">Visibility</Label>
					<select
						id="visibility"
						bind:value={visibility}
						disabled={updateMutation.isPending}
						class="border-input h-10 w-full rounded-md border bg-background px-3 text-sm focus:ring-2 focus:ring-primary focus:outline-none disabled:opacity-50"
					>
						<option value="public">Public</option>
						<option value="unlisted">Unlisted</option>
						<option value="private">Private</option>
					</select>
				</div>
			</div>

			<div class="space-y-1.5">
				<Label for="tags">Tags (comma separated)</Label>
				<Input
					id="tags"
					bind:value={tags}
					placeholder="e.g. gaming, funny, stream"
					disabled={updateMutation.isPending}
				/>
			</div>

			<div class="space-y-1.5">
				<Label>Custom Thumbnail</Label>
				<div class="flex items-center gap-4">
					<input
						type="file"
						id="thumbnail"
						accept="image/*"
						class="hidden"
						onchange={(e) => {
							const target = e.target as HTMLInputElement;
							if (target.files?.length) thumbnailFile = target.files[0];
						}}
						disabled={updateMutation.isPending}
					/>
					<Button
						type="button"
						variant="outline"
						onclick={() => document.getElementById('thumbnail')?.click()}
						disabled={updateMutation.isPending}
					>
						<ImageIcon class="mr-2 h-4 w-4" />
						{thumbnailFile ? thumbnailFile.name : 'Choose Image'}
					</Button>
				</div>
			</div>

			<div class="flex items-center space-x-2 pt-2">
				<input
					type="checkbox"
					id="isAdult"
					bind:checked={isAdult}
					disabled={updateMutation.isPending}
					class="border-input h-4 w-4 rounded text-primary focus:ring-primary disabled:opacity-50"
				/>
				<Label for="isAdult" class="cursor-pointer">Contains Adult Content</Label>
			</div>

			<div class="flex justify-end gap-3 border-t pt-4">
				<Button
					type="button"
					variant="outline"
					onclick={() => (open = false)}
					disabled={updateMutation.isPending}
				>
					Cancel
				</Button>
				<Button type="submit" disabled={updateMutation.isPending}>
					{#if updateMutation.isPending}
						<Loader2 class="mr-2 h-4 w-4 animate-spin" /> Saving...
					{:else}
						Save Changes
					{/if}
				</Button>
			</div>
		</form>
	</DialogContent>
</Dialog>
