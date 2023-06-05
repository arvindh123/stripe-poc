'use client';
import { Toast } from 'flowbite-react';
import { RxCross1 } from 'react-icons/rx'
import { useRouter } from 'next/navigation'

// only import what you want to use
import {
    Button,
    Checkbox,
    FileInput,
    Label,
    Radio,
    RangeSlider,
    Select,
    Textarea,
    TextInput,
    ToggleSwitch,
} from 'flowbite-react';
import { useState } from 'react';

export default function Create() {
    const [name, setName] = useState("")
    const [email, setEmail] = useState("")
    const [errMsg, setErrMsg] = useState("")
    const router = useRouter();
    const handleSubmit = async (e: React.FormEvent<HTMLFormElement>): Promise<void> => {
        e.preventDefault();
        fetch("http://localhost:8080/organization/create", {
            body: JSON.stringify({ name: `${name}`, email: `${email}` }),
            method: "post",
            headers: {
                "content-type": "application/json",
            },
        }).then(async (result) => {
            try {
                if (result.status == 200) {
                    router.push('/organization')
                } else {
                    const body = await result.text()
                    setErrMsg(body)
                }
            } catch (err) {
                setErrMsg((err as Error).message)
            }
        });
    };

    return (
        <form className="flex max-w-md flex-col gap-4 " onSubmit={handleSubmit} >
            <h1 className="font-extrabold text-transparent text-6xl bg-clip-text bg-gradient-to-r from-purple-400 to-pink-600" >
                Create New Organization
            </h1>
            <div>
                <div className="mb-2 block">
                    <Label
                        className='text-white'
                        text-white
                        htmlFor="orgname"
                        value="Organization Name"
                    />
                </div>
                <TextInput
                    id="orgname"
                    placeholder="ACME Corp"
                    required
                    shadow
                    type="text"
                    onChange={(e) => setName(e.target.value)}
                />
            </div>
            <div>
                <div className="mb-2 block">
                    <Label
                        className='text-white'
                        htmlFor="email12"
                        value="Organization Email"
                    />
                </div>
                <TextInput
                    id="email12"
                    placeholder="admin@acme.com"
                    required
                    shadow
                    type="email"
                    onChange={(e) => setEmail(e.target.value)}
                />
            </div>


            <Button type="submit">
                Create Organization
            </Button>
            {
                errMsg !== "" ?
                    <Toast>
                        <div className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-red-100 text-red-500 dark:bg-red-800 dark:text-red-200">
                            <RxCross1 className="h-5 w-5" />
                        </div>
                        <div className="ml-3 text-sm font-normal">
                            Could not create Organization , Error : { errMsg }
                        </div>
                        <Toast.Toggle onClick={() => setErrMsg("")}/>
                    </Toast>
            : null
            }

        </form>
    )
}