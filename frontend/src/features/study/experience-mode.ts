"use client";
import {useEffect,useState} from "react";
import {taxonomyRequest} from "@/lib/taxonomy/client";
export function useExperienceMode(path:string){const[mode,setMode]=useState<"legacy"|"topics"|null>(null);useEffect(()=>{let live=true,generation=0;const refresh=()=>{const g=++generation;setMode(null);void taxonomyRequest<{mode:"legacy"|"topics"}>({kind:"readExperience"}).then(r=>{if(live&&g===generation)setMode(r.ok?r.data.mode:null)})};refresh();window.addEventListener("focus",refresh);return()=>{live=false;generation++;window.removeEventListener("focus",refresh)}},[path]);return mode}
