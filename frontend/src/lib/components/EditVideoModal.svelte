<script lang="ts">
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogTitle,
		DialogDescription
	} from '#lib/components/ui/dialog';
	import { Input } from '#lib/components/ui/input';
	import { Label } from '#lib/components/ui/label';
	import { Textarea } from '#lib/components/ui/textarea';
	import { Button } from '#lib/components/ui/button';
	import { fetchApi } from '#lib/api';
	import { createMutation, useQueryClient } from '@tanstack/svelte-query';
	import {
		Loader2,
		Image as ImageIcon,
		Edit,
		Hash,
		X,
		Plus,
		ShieldAlert,
		CheckCircle2
	} from 'lucide-svelte';

	let {
		video,
		open = $bindable(false),
		onClose
	} = $props<{ video: any; open: boolean; onClose?: () => void }>();

	const queryClient = useQueryClient();

	let title = $state('');
	let description = $state('');
	let category = $state('Other');
	let tags = $state<string[]>([]);
	let tagInput = $state('');
	let visibility = $state('public');
	let isAdult = $state(false);

	let thumbnailFile: File | null = $state(null);
	let thumbnailPreview = $state('');
	let error = $state('');
	let successMsg = $state('');

	const suggestedTags = [
		'gaming',
		'tutorial',
		'music',
		'tech',
		'vlog',
		'entertainment',
		'coding',
		'review',
		'comedy',
		'education'
	];

	function addTag(rawTag: string) {
		const clean = rawTag.trim().replace(/^#+/, '').toLowerCase();
		if (!clean) return;
		if (tags.length >= 15) {
			error = 'Maximum 15 tags allowed per video.';
			return;
		}
		if (!tags.includes(clean)) {
			tags = [...tags, clean];
			error = '';
		}
		tagInput = '';
	}

	function removeTag(tagToRemove: string) {
		tags = tags.filter((t) => t !== tagToRemove);
	}

	function handleTagKeyDown(e: KeyboardEvent) {
		if (e.key === 'Enter' || e.key === ',') {
			e.preventDefault();
			addTag(tagInput);
		} else if (e.key === 'Backspace' && tagInput === '' && tags.length > 0) {
			removeTag(tags[tags.length - 1]);
		}
	}

	function handleThumbnailSelect(e: Event) {
		const target = e.target as HTMLInputElement;
		if (target.files && target.files.length > 0) {
			thumbnailFile = target.files[0];
			try {
				thumbnailPreview = URL.createObjectURL(thumbnailFile);
			} catch {
				thumbnailPreview = '';
			}
		}
	}

	// Sync local state when video prop changes or modal opens
	$effect(() => {
		if (open && video) {
			title = video.title || '';
			description = video.description || '';
			category = video.category || 'Other';
			if (Array.isArray(video.tags)) {
				tags = [...video.tags];
			} else if (typeof video.tags === 'string' && video.tags) {
				tags = video.tags
					.split(',')
					.map((t: string) => t.trim().replace(/^#+/, ''))
					.filter(Boolean);
			} else {
				tags = [];
			}
			tagInput = '';
			visibility = video.visibility || 'public';
			isAdult = !!video.isAdult;
			thumbnailFile = null;
			thumbnailPreview = video.thumbnailUrl || '';
			error = '';
			successMsg = '';
		}
	});

	const updateMutation = createMutation(
		() => ({
			mutationFn: async () => {
				const videoId = video.id || video.fileId;
				// Update metadata
				const metaReq = fetchApi(`/videos/${videoId}`, {
					method: 'PUT',
					body: JSON.stringify({
						title: title.trim(),
						description: description.trim(),
						category,
						visibility,
						isAdult,
						tags
					})
				});

				// Update thumbnail if provided
				if (thumbnailFile) {
					const formData = new FormData();
					formData.append('thumbnailFile', thumbnailFile);
					const thumbReq = fetchApi(`/videos/${videoId}/update-thumbnail`, {
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
				if (video?.id) queryClient.invalidateQueries({ queryKey: ['video', video.id] });
				if (video?.fileId) queryClient.invalidateQueries({ queryKey: ['video', video.fileId] });
				successMsg = 'Video updated successfully!';
				setTimeout(() => {
					open = false;
					if (onClose) onClose();
				}, 600);
			},
			onError: (err: any) => {
				error = err.message || 'Failed to update video metadata.';
			}
		}),
		() => queryClient
	);

	function handleSubmit(e: Event) {
		e.preventDefault();
		if (!title.trim()) {
			error = 'Title is required.';
			return;
		}
		updateMutation.mutate();
	}
</script>

<Dialog bind:open>
	<DialogContent class="max-h-[90vh] overflow-y-auto rounded-2xl sm:max-w-[620px]">
		<DialogHeader class="border-b pb-4">
			<div class="flex items-center gap-2.5">
				<div class="flex h-9 w-9 items-center justify-center rounded-xl bg-primary/10 text-primary">
					<Edit class="h-5 w-5" />
				</div>
				<div>
					<DialogTitle class="text-lg font-bold">Edit Video Details</DialogTitle>
					<DialogDescription class="text-muted-foreground mt-0.5 text-xs">
						Update video metadata, category, tags, and audience settings in-place.
					</DialogDescription>
				</div>
			</div>
		</DialogHeader>

		{#if error}
			<div
				class="bg-destructive/10 text-destructive mt-4 flex items-center justify-between rounded-xl p-3 text-xs font-semibold"
			>
				<span>{error}</span>
				<button type="button" onclick={() => (error = '')}><X class="h-3.5 w-3.5" /></button>
			</div>
		{/if}

		{#if successMsg}
			<div
				class="mt-4 flex items-center gap-2 rounded-xl bg-emerald-500/10 p-3 text-xs font-semibold text-emerald-600 dark:text-emerald-400"
			>
				<CheckCircle2 class="h-4 w-4" />
				<span>{successMsg}</span>
			</div>
		{/if}

		<form onsubmit={handleSubmit} class="space-y-5 pt-3 pb-2">
			<!-- Title -->
			<div class="space-y-1.5">
				<Label for="title" class="text-xs font-semibold">Title <span class="text-destructive">*</span></Label>
				<Input
					id="title"
					bind:value={title}
					required
					disabled={updateMutation.isPending}
					placeholder="Video title"
					class="font-medium"
				/>
			</div>

			<!-- Description -->
			<div class="space-y-1.5">
				<Label for="description" class="text-xs font-semibold">Description</Label>
				<Textarea
					id="description"
					bind:value={description}
					rows={3}
					disabled={updateMutation.isPending}
					placeholder="Tell viewers about your video..."
					class="resize-y text-xs leading-relaxed"
				/>
			</div>

			<!-- Category & Visibility -->
			<div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
				<div class="space-y-1.5">
					<Label for="category" class="text-xs font-semibold">Category</Label>
					<select
						id="category"
						bind:value={category}
						disabled={updateMutation.isPending}
						class="border-input h-10 w-full rounded-md border bg-background px-3 text-xs font-medium focus:ring-2 focus:ring-primary focus:outline-none disabled:opacity-50"
					>
						<option value="Other">Other</option>
						<option value="Gaming">Gaming</option>
						<option value="Education">Education</option>
						<option value="Entertainment">Entertainment</option>
						<option value="Music">Music</option>
						<option value="Science">Science & Tech</option>
						<option value="Vlog">Vlog</option>
						<option value="Comedy">Comedy</option>
					</select>
				</div>
				<div class="space-y-1.5">
					<Label for="visibility" class="text-xs font-semibold">Visibility</Label>
					<select
						id="visibility"
						bind:value={visibility}
						disabled={updateMutation.isPending}
						class="border-input h-10 w-full rounded-md border bg-background px-3 text-xs font-medium focus:ring-2 focus:ring-primary focus:outline-none disabled:opacity-50"
					>
						<option value="public">Public (Everyone can watch)</option>
						<option value="unlisted">Unlisted (Anyone with link)</option>
						<option value="private">Private (Only you)</option>
					</select>
				</div>
			</div>

			<!-- Interactive Tag Chips -->
			<div class="space-y-2">
				<div class="flex items-center justify-between">
					<Label class="flex items-center gap-1.5 text-xs font-semibold">
						<Hash class="h-3.5 w-3.5 text-primary" />
						Tags
					</Label>
					<span class="text-muted-foreground text-[11px]">{tags.length}/15 tags</span>
				</div>

				<div
					class="border-input focus-within:ring-2 focus-within:ring-primary focus-within:border-primary flex min-h-[42px] w-full flex-wrap items-center gap-1.5 rounded-xl border bg-background/50 p-2 text-xs transition-all"
				>
					{#each tags as tag (tag)}
						<span
							class="inline-flex items-center gap-1 rounded-lg border border-primary/30 bg-primary/10 px-2 py-0.5 text-xs font-semibold text-primary transition-all animate-in zoom-in-95 duration-150"
						>
							<span>#{tag}</span>
							<button
								type="button"
								onclick={() => removeTag(tag)}
								disabled={updateMutation.isPending}
								class="text-primary/70 hover:text-primary hover:bg-primary/20 -mr-0.5 ml-0.5 rounded-full p-0.5 transition-colors"
								title={`Remove #${tag}`}
							>
								<X class="h-3 w-3" />
							</button>
						</span>
					{/each}

					<input
						type="text"
						bind:value={tagInput}
						onkeydown={handleTagKeyDown}
						placeholder={tags.length === 0 ? 'Type tag and press Enter...' : 'Add tag...'}
						disabled={updateMutation.isPending || tags.length >= 15}
						class="min-w-[110px] flex-1 bg-transparent px-2 py-1 text-xs text-foreground placeholder:text-muted-foreground/60 focus:outline-none disabled:opacity-50"
					/>

					{#if tagInput.trim().length > 0}
						<Button
							type="button"
							variant="ghost"
							size="sm"
							class="h-6 gap-1 rounded-md px-2 text-[11px] font-semibold text-primary hover:bg-primary/10"
							onclick={() => addTag(tagInput)}
						>
							<Plus class="h-3 w-3" /> Add
						</Button>
					{/if}
				</div>

				<!-- Suggested Quick Tags -->
				<div class="flex flex-wrap items-center gap-1 pt-0.5">
					<span class="text-muted-foreground text-[10px] font-medium">Quick add:</span>
					{#each suggestedTags as suggestion}
						{#if !tags.includes(suggestion)}
							<button
								type="button"
								onclick={() => addTag(suggestion)}
								disabled={updateMutation.isPending || tags.length >= 15}
								class="border-border/70 bg-muted/40 text-muted-foreground hover:border-primary/50 hover:bg-primary/10 hover:text-primary rounded-md border px-1.5 py-0.5 text-[10px] font-medium transition-colors disabled:opacity-40"
							>
								+{suggestion}
							</button>
						{/if}
					{/each}
				</div>
			</div>

			<!-- 18+ Adult Content Toggle Card -->
			<div class="border-border/80 bg-muted/20 rounded-xl border p-3.5 transition-colors">
				<label for="isAdultModal" class="flex cursor-pointer items-start gap-3">
					<input
						type="checkbox"
						id="isAdultModal"
						bind:checked={isAdult}
						disabled={updateMutation.isPending}
						class="border-input text-primary focus:ring-primary mt-1 h-4 w-4 rounded disabled:opacity-50"
					/>
					<div class="flex-1 space-y-0.5">
						<div class="flex items-center gap-2">
							<span class="text-xs font-semibold text-foreground">18+ Mature Content</span>
							{#if isAdult}
								<span
									class="inline-flex items-center gap-1 rounded border border-amber-500/30 bg-amber-500/15 px-1.5 py-0.2 font-mono text-[10px] font-bold text-amber-500"
								>
									<ShieldAlert class="h-3 w-3" /> RESTRICTED
								</span>
							{/if}
						</div>
						<p class="text-muted-foreground text-[11px] leading-normal">
							When enabled, viewers must be logged in and confirm 18+ mode in order to watch this video.
						</p>
					</div>
				</label>
			</div>

			<!-- Custom Thumbnail -->
			<div class="space-y-2 border-t pt-3">
				<Label class="text-xs font-semibold">Custom Thumbnail</Label>
				<div class="flex items-center gap-4">
					{#if thumbnailPreview}
						<div class="bg-muted relative h-14 w-24 shrink-0 overflow-hidden rounded-lg border shadow-sm">
							<img src={thumbnailPreview} alt="Thumbnail preview" class="h-full w-full object-cover" />
						</div>
					{/if}
					<input
						type="file"
						id="modalThumbnail"
						accept="image/*"
						class="hidden"
						onchange={handleThumbnailSelect}
						disabled={updateMutation.isPending}
					/>
					<Button
						type="button"
						variant="outline"
						size="sm"
						onclick={() => document.getElementById('modalThumbnail')?.click()}
						disabled={updateMutation.isPending}
						class="h-9 gap-1.5 text-xs font-medium"
					>
						<ImageIcon class="text-primary h-4 w-4" />
						{thumbnailFile ? thumbnailFile.name : 'Choose New Thumbnail'}
					</Button>
				</div>
			</div>

			<!-- Footer Buttons -->
			<div class="flex justify-end gap-3 border-t pt-4">
				<Button
					type="button"
					variant="outline"
					size="sm"
					onclick={() => (open = false)}
					disabled={updateMutation.isPending}
				>
					Cancel
				</Button>
				<Button
					type="submit"
					size="sm"
					disabled={updateMutation.isPending}
					class="font-semibold shadow-sm"
				>
					{#if updateMutation.isPending}
						<Loader2 class="mr-2 h-4 w-4 animate-spin" /> Saving Changes...
					{:else}
						Save Changes
					{/if}
				</Button>
			</div>
		</form>
	</DialogContent>
</Dialog>
