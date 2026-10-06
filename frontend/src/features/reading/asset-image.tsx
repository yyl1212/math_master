"use client";
import {useUiI18n} from "@/lib/i18n/provider";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import { useState, useEffect } from "react";
import type { AssetView } from "@/lib/api/types";
import type {AssetScope} from "@/lib/content/types";
import {contentRouteRequest} from "@/lib/content/schemas";
import styles from "@/styles/reading.module.css";
export function AssetImage({ asset, alt, assetScope }: { asset: AssetView; alt: string; assetScope?:AssetScope }) {
  const {t}=useUiI18n();
  const src=assetScope?contentRouteRequest({kind:assetScope.kind==="draft"?"readDraftAsset":"readSubmissionAsset",id:assetScope.id,sha:asset.sha256})?.path:"/api/v1/assets/"+asset.sha256;
  const [failed, setFailed] = useState(false);
  useEffect(()=>setFailed(false),[src]);
  return (
    <span className={styles.figure}>
      {failed || !src ? (
        <span className={styles.figureError} role="status"><UiText notice={uiMessage("asset-image.illustration.is.temporarily.unavailable.b89403",{})}/></span>
      ) : (
        <img
          src={src}
          alt={alt || t("audit.illustration",{})}
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
