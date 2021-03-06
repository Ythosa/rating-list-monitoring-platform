import React, { useContext, useEffect, useState } from 'react'
import { AuthContext, AuthContextType } from '../../context/auth.context'
import { Avatar, createStyles, makeStyles, Theme } from '@material-ui/core'
import profileIcon from './profile.png'
import UserService from '../../services/user-service'
import { Link } from 'react-router-dom'
import Loader from '../loader'

import './header.css'

const useStyles = makeStyles((theme: Theme) =>
    createStyles({
        large: {
            width: theme.spacing(6),
            height: theme.spacing(6),
        },
    }),
)

export const Header = () => {
    const classes = useStyles()
    const authContext = useContext<AuthContextType>(AuthContext)

    const [ loading, setLoading ] = useState<boolean>(true)
    const [ username, setUsername ] = useState<string>('')

    useEffect(() => {
        const userService = new UserService(authContext.accessToken!!)
        userService.getUsername()
            .then(u => {
                setLoading(false)
                setUsername(u)
            })
            .catch(e => {
                console.log(e)
                authContext.logout()
            })
    }, [ authContext ])

    const loadingBanner = loading ? <Loader/> : null
    const content = !loading ? (
        <React.Fragment>
            <Link to="/profile" className="header-username-profile">
                <Avatar alt={username} src={profileIcon} className={classes.large}/>
                <span className="header-username">{username}</span>
            </Link>
            <Link to="#" className="header-nav-link">ВУЗЫ</Link>
            <Link to="/results" className="header-nav-link">Программы</Link>
            <Link to="#" onClick={authContext.logout} className="header-nav-link">Выйти</Link>
        </React.Fragment>
    ) : null

    return (
        <div className="header-wrapper">
            {loadingBanner}
            {content}
        </div>
    )
}
