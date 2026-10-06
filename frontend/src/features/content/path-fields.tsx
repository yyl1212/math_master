"use client";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage,uiValue} from "@/lib/i18n/format";
import {useUiI18n} from "@/lib/i18n/provider";
import type { Path } from "@/lib/content/types";
import { TextField, NumberField, TextList, RefFields, Rows } from "./field-controls";
export const emptyPath = (): Path => ({ id: "new-path", version: 1, domainIds: [], title: "", titleZh: "", nodes: [] });
export function PathFields({ value, label, onChange }: {
    value: Path;
    label: string;
    onChange: (v: Path) => void;
}) {
 const {t}=useUiI18n();
 return <fieldset><legend>{label}</legend><TextField label={t("path-fields.value.id.0ad111",{v0:uiValue(label)})} value={value.id} onChange={id => onChange({ ...value, id })}/><NumberField label={t("path-fields.value.version.f8a7ae",{v0:uiValue(label)})} value={value.version} onChange={version => onChange({ ...value, version })}/><TextField label={t("path-fields.value.title.ed4f63",{v0:uiValue(label)})} value={value.title} onChange={title => onChange({ ...value, title })}/><TextField label={t("path-fields.value.chinese.title.74d780",{v0:uiValue(label)})} value={value.titleZh} onChange={titleZh => onChange({ ...value, titleZh })}/><TextList label={t("path-fields.value.domain.id.066d9c",{v0:uiValue(label)})} items={value.domainIds} onChange={domainIds => onChange({ ...value, domainIds })}/><Rows label={t("path-fields.value.node.ce3c13",{v0:uiValue(label)})} items={value.nodes} onChange={nodes => onChange({ ...value, nodes })} create={() => ({ id: "new-knowledge", version: 1 })}>{(v, i, change) => <RefFields label={t("path-fields.value.node.value.5f4a60",{v0:uiValue(label),v1:uiValue(i + 1)})} value={v} onChange={change}/>}</Rows></fieldset>; }
