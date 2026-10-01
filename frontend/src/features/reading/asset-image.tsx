"use client";
import { useState, useEffect } from "react";
import type { AssetView } from "@/lib/api/types";
import type {AssetScope} from "@/lib/content/types";
import {contentRouteRequest} from "@/lib/content/schemas";
import styles from "@/styles/reading.module.css";
export function AssetImage({ asset, alt, assetScope }: { asset: AssetView; alt: string; assetScope?:AssetScope }) {
  const src=assetScope?contentRouteRequest({kind:assetScope.kind==="draft"?"readDraftAsset":"readSubmissionAsset",id:assetScope.id,sha:asset.sha256})?.path:"/api/v1/assets/"+asset.sha256;
  const [failed, setFailed] = useState(false);
  useEffect(()=>setFailed(false),[src]);
  return (
    <span className={styles.figure}>
      {failed || !src ? (
        <span className={styles.figureError} role="status">
          Illustration is temporarily unavailable.
        </span>
      ) : (
        <img
          src={src}
          alt={alt || "Mathematical illustration"}
          loading="lazy"
          onError={() => setFailed(true)}
        />
      )}
      <span className={styles.figureCaption}>
        {asset.author} · <span>{asset.license}</span>
        {asset.attribution && <span>{asset.attribution}</span>}
      </span>
    </span>
  );
}
