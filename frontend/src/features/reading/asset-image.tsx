"use client";
import { useState } from "react";
import type { AssetView } from "@/lib/api/types";
import styles from "@/styles/reading.module.css";
export function AssetImage({ asset, alt }: { asset: AssetView; alt: string }) {
  const [failed, setFailed] = useState(false);
  return (
    <span className={styles.figure}>
      {failed ? (
        <span className={styles.figureError} role="status">
          Illustration is temporarily unavailable.
        </span>
      ) : (
        <img
          src={"/api/v1/assets/" + asset.sha256}
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
