'use client';
import { Card, Button } from 'flowbite-react';
import Plan from '../components/Plan';
import { MdOutlineKeyboardBackspace } from 'react-icons/md'
import { useRouter } from 'next/navigation'
import { TOrg } from '../../interface/all'
import { cancelSub, getSub, getOrg } from '../../backendAPI/getAllOrg'
import { useState } from 'react'
import { BiRefresh } from 'react-icons/bi'
import { Toast } from 'flowbite-react';
import { RxCross1 } from 'react-icons/rx'

export default function View({ org }: { org: TOrg }) {
    const router = useRouter()
    const [msg, setMsg] = useState("")
    const [isProcessing, setIsProcessing] = useState(false)
    const handleCancel = async (org: TOrg) => {
        setIsProcessing(true)
        await cancelSub(org.id)
        router.push(`/organization/${org.id}`)
        return
    }
    const handleCompletePayment = async (org: TOrg) => {
        const res = await getSub(org.id)
        if ((res?.clientSecret) && (res?.clientSecret !== "")) {
            setIsProcessing(true)
            router.push(`/checkout?payment=${res?.clientSecret}&id=${org.id}`)
            return
        }
    }
    const handleUpdateSub = async () => {
        setIsProcessing(true)
        try {

            const res = await getSub(org.id)
            // console.log(res)
            if (res?.subscriptionId) {
                router.refresh()
            }
        } catch (err) {
            console.log((err as Error)?.message)
        }

        setIsProcessing(false)
    }

    const handleChossePlan = async (org: TOrg) => {
        setIsProcessing(true)
        router.push(`plans?orgID=${org.id}`)
        return
    }
    return (
        <div>

            <Card className='w-24 min-w-fit'>
                <div className="flex justify-between">
                    <button className="flex-initial text-2xl  text-gray-900 dark:text-white" onClick={() => { router.push("/organization") }}>
                        <MdOutlineKeyboardBackspace />
                    </button>
                    <button className="flex-initial font-bold text-2xl text-gray-700 dark:text-gray-400" onClick={() => { handleUpdateSub() }}>
                        <BiRefresh />
                    </button>
                </div>

                <h5 className="text-2xl font-bold tracking-tight text-gray-900 dark:text-white">
                    <p>
                        Organization: {org.name}
                    </p>
                </h5>
                <p className="font-normal text-gray-700 dark:text-gray-400">
                    E-mail: {org.email}
                </p>
                <p className="font-normal text-gray-700 dark:text-gray-400">
                    Customer ID:  {org.stripe_id}
                </p>
                <p className="font-normal text-gray-700 dark:text-gray-400">
                    Current Subscription:  {org.stripe_sub}
                </p>
                <p className="font-normal text-gray-700 dark:text-gray-400">
                    Current Subscription Status:  {org.sub_status}
                </p>
                <>
                    {

                        org.plans !== null ?
                            org.plans.map((item, index) => {
                                return (
                                    <>
                                        <Card key={index}>
                                            <h5 className="font-bold tracking-tight text-gray-900 dark:text-white">
                                                <p>
                                                    Product Name:  {item.product.name}
                                                </p>
                                            </h5>
                                            <p className="font-normal text-gray-700 dark:text-gray-400">
                                                Product Description:  {item.product.description}
                                            </p>
                                            <p className="font-normal text-gray-700 dark:text-gray-400">
                                                Price :  {item.amount}
                                            </p>
                                            <p className="font-normal text-gray-700 dark:text-gray-400">
                                                Plan ID:  {item.id}
                                            </p>

                                            <p className="font-normal text-gray-700 dark:text-gray-400">
                                                Plan Status:  {item.active ? "Active" : "Inactive"}
                                            </p>

                                            <p className="font-normal text-gray-700 dark:text-gray-400">
                                                Plan Quantity:  {item.quantity}
                                            </p>
                                        </Card>
                                    </>
                                )
                            })
                            : null
                    }
                </>

                {
                    org.stripe_sub.trim() !== ""
                        ?
                        <>
                            {
                                (["incomplete", "past_due"].find(item => item === org.sub_status)) ?
                                    <Button gradientDuoTone="greenToBlue" disabled={isProcessing} onClick={() => handleCompletePayment(org)}>
                                        <p>
                                            Complete Subscription Payment
                                        </p>
                                    </Button>
                                    : null
                            }
                            <Button gradientDuoTone="pinkToOrange" disabled={isProcessing} onClick={() => handleCancel(org)}>
                                <p>
                                    Cancel Subscription
                                </p>
                            </Button>
                            <Button gradientDuoTone="purpleToBlue" disabled={isProcessing} onClick={() => { handleChossePlan(org) }}>
                                <p>
                                    Change Subscription Plan
                                </p>
                            </Button>
                        </>
                        :
                        <>
                            <Button gradientDuoTone="purpleToBlue" disabled={isProcessing} onClick={() => { handleChossePlan(org) }}>
                                <p>
                                    Choose a Plan
                                </p>
                            </Button>
                        </>

                }

            </Card>
            {
                msg !== "" ?
                    <Toast>
                        <div className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-red-100 text-red-500 dark:bg-red-800 dark:text-red-200">
                            <RxCross1 className="h-5 w-5" />
                        </div>
                        <div className="ml-3 text-sm font-normal">
                            {msg}
                        </div>
                        <Toast.Toggle onClick={() => setMsg("")} />
                    </Toast>
                    : null
            }
        </div>

    )
}