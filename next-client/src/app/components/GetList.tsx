'use client';
import { TOrg } from "../../interface/all"
import { Table } from 'flowbite-react';

export default function GetList({ allOrg }: { allOrg: TOrg[] }) {
    return (
        <Table>
            <Table.Head>
                <Table.HeadCell>
                    ID
                </Table.HeadCell>
                <Table.HeadCell>
                    Name
                </Table.HeadCell>
                <Table.HeadCell>
                    Email
                </Table.HeadCell>
                <Table.HeadCell>
                    Stripe ID
                </Table.HeadCell>
                <Table.HeadCell>
                    Stripe Subscription
                </Table.HeadCell>
                <Table.HeadCell>
                    Subscription Status
                </Table.HeadCell>
                <Table.HeadCell>
                    <span className="sr-only">
                        Edit
                    </span>
                </Table.HeadCell>
            </Table.Head>
            <Table.Body className="divide-y">

                {
                    allOrg.map((org,index) => {
                        return(
                        <Table.Row className="bg-white dark:border-gray-700 dark:bg-gray-800" key={index}>
                            <Table.Cell className="whitespace-nowrap font-medium text-gray-900 dark:text-white">
                                {org.id.toString()}
                            </Table.Cell>
                            <Table.Cell>
                                {org.name}
                            </Table.Cell>
                            <Table.Cell>
                                {org.email}
                            </Table.Cell>
                            <Table.Cell>
                                {org.stripe_id}
                            </Table.Cell>
                            <Table.Cell>
                                {org.stripe_sub}
                            </Table.Cell>
                            <Table.Cell>
                                {org.sub_status}
                            </Table.Cell>
                            <Table.Cell>
                                <a
                                    className="font-medium text-cyan-600 hover:underline dark:text-cyan-500"
                                    href={'/organization/' + org.id}
                                >
                                    <p>
                                        View
                                    </p>
                                </a>
                            </Table.Cell>
                        </Table.Row>
                        )
                    })
                }


            </Table.Body>
        </Table>
    )
}
