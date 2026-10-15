<script lang="ts">
	import { ShieldAlert, AlertTriangle, CheckCircle, X } from 'lucide-svelte';
	import { Button } from '#lib/components/ui/button';
	import { grantAdultConsent, setContentMode } from '#lib/consent.svelte';

	let {
		isOpen = $bindable(false),
		onConsent,
		onCancel
	}: {
		isOpen: boolean;
		onConsent?: () => void;
		onCancel?: () => void;
	} = $props();

	function handleConfirm() {
		grantAdultConsent();
		setContentMode(true);
		isOpen = false;
		if (onConsent) {
			onConsent();
		}
	}

	function handleClose() {
		isOpen = false;
		if (onCancel) {
			onCancel();
		}
	}
</script>

{#if isOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-md animate-in fade-in-0 duration-200"
		role="dialog"
		aria-modal="true"
		aria-labelledby="consent-title"
	>
		<div
			class="relative w-full max-w-lg rounded-2xl border border-amber-500/30 bg-background/95 p-6 shadow-2xl backdrop-blur-xl sm:p-8"
		>
			<!-- Close Icon Button -->
			<button
				type="button"
				class="hover:bg-muted text-muted-foreground hover:text-foreground absolute top-4 right-4 rounded-full p-1.5 transition-colors"
				onclick={handleClose}
				aria-label="Close dialog"
			>
				<X class="h-5 w-5" />
			</button>

			<div class="flex flex-col items-center text-center">
				<!-- Warning Badge -->
				<div
					class="mb-4 flex h-16 w-16 items-center justify-center rounded-2xl bg-amber-500/10 text-amber-500 border border-amber-500/30 shadow-inner"
				>
					<ShieldAlert class="h-8 w-8" />
				</div>

				<div class="inline-flex items-center gap-1.5 rounded-full bg-amber-500/15 px-3 py-1 text-xs font-semibold text-amber-500 dark:text-amber-400 mb-2">
					<span class="inline-block h-2 w-2 rounded-full bg-amber-500 animate-pulse"></span>
					Age-Restricted Content (18+)
				</div>

				<h2 id="consent-title" class="text-xl font-bold tracking-tight text-foreground sm:text-2xl">
					Adult Content Consent
				</h2>

				<p class="text-muted-foreground mt-3 text-sm leading-relaxed sm:text-base">
					This section contains sexually explicit material intended solely for adults aged 18 and older (or legal age of majority in your jurisdiction).
				</p>

				<div class="mt-4 rounded-xl border border-border/60 bg-muted/30 p-3.5 text-left text-xs text-muted-foreground space-y-1.5 w-full">
					<div class="flex items-start gap-2">
						<CheckCircle class="h-4 w-4 shrink-0 text-emerald-500 mt-0.5" />
						<span>Adult content is kept completely isolated from general feeds and search.</span>
					</div>
					<div class="flex items-start gap-2">
						<CheckCircle class="h-4 w-4 shrink-0 text-emerald-500 mt-0.5" />
						<span>You can revoke this consent or return to the safe general feed at any time.</span>
					</div>
				</div>

				<div class="mt-6 flex w-full flex-col-reverse gap-3 sm:flex-row sm:justify-end">
					<Button
						type="button"
						variant="outline"
						class="w-full rounded-xl sm:w-auto"
						onclick={handleClose}
					>
						Keep Me Safe (Exit)
					</Button>
					<Button
						type="button"
						class="w-full rounded-xl bg-amber-600 hover:bg-amber-700 text-white shadow-lg shadow-amber-600/20 sm:w-auto font-semibold"
						onclick={handleConfirm}
					>
						I am 18 or Older & Consent
					</Button>
				</div>
			</div>
		</div>
	</div>
{/if}
