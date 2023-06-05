"use client"
import { getOrg, getPlans } from "@/backendAPI/getAllOrg"
import { TOrg, TPrice } from "../../interface/all"
import { useSearchParams } from 'next/navigation';
import Plan from "../components/Plan"

export default async function Subscribe()  {
    const searchParams = useSearchParams();
    const orgID = searchParams.get("orgID")

    const org = await getOrg(orgID)
    const plans = await getPlans()
    return (
        <Plan org={(org as TOrg)} plans={(plans as TPrice[])}/>
    )
}

