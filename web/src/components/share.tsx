import { useEffect, useState } from "react";
import { amount } from "../meta";
import type { Listing } from "../types";

// shareUrl is the public link for a listing. It opens the listing for
// anyone, and chat apps show a preview with its photo.
export const shareUrl = (id: string) => `${window.location.origin}/l/${encodeURIComponent(id)}`;

// ShareButton shares a listing's link through the phone's own share sheet
// (WhatsApp, SMS, Facebook, email...) where there is one, and otherwise
// copies the link so it can be pasted anywhere.
export function ShareButton({ listing, className = "btn btn-small" }: { listing: Listing; className?: string }) {
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (!copied) return;
    const t = setTimeout(() => setCopied(false), 2400);
    return () => clearTimeout(t);
  }, [copied]);

  async function share() {
    const url = shareUrl(listing.id);
    const text = `Free to a good home: ${listing.item}, ${amount(listing)} in ${listing.area}.`;
    if (navigator.share) {
      try {
        await navigator.share({ title: listing.item, text, url });
        return;
      } catch (e) {
        // The person closed the share sheet: nothing to do.
        if ((e as Error).name === "AbortError") return;
      }
    }
    try {
      await navigator.clipboard.writeText(url);
      setCopied(true);
    } catch {
      // No clipboard access (an old browser or plain http): show the link.
      window.prompt("Copy this link to share the listing:", url);
    }
  }

  return (
    <button type="button" className={className} onClick={share}>
      {copied ? "Link copied" : "Share"}
    </button>
  );
}
