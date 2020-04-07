-- Do not manually edit this file, it was auto-generated by dillonkearns/elm-graphql
-- https://github.com/dillonkearns/elm-graphql


module DetQL.Enum.Users_select_column exposing (..)

import Json.Decode as Decode exposing (Decoder)


{-| select columns of table "users"

  - Active - column name
  - Admin - column name
  - Id - column name
  - Username - column name

-}
type Users_select_column
    = Active
    | Admin
    | Id
    | Username


list : List Users_select_column
list =
    [ Active, Admin, Id, Username ]


decoder : Decoder Users_select_column
decoder =
    Decode.string
        |> Decode.andThen
            (\string ->
                case string of
                    "active" ->
                        Decode.succeed Active

                    "admin" ->
                        Decode.succeed Admin

                    "id" ->
                        Decode.succeed Id

                    "username" ->
                        Decode.succeed Username

                    _ ->
                        Decode.fail ("Invalid Users_select_column type, " ++ string ++ " try re-running the @dillonkearns/elm-graphql CLI ")
            )


{-| Convert from the union type representating the Enum to a string that the GraphQL server will recognize.
-}
toString : Users_select_column -> String
toString enum =
    case enum of
        Active ->
            "active"

        Admin ->
            "admin"

        Id ->
            "id"

        Username ->
            "username"


{-| Convert from a String representation to an elm representation enum.
This is the inverse of the Enum `toString` function. So you can call `toString` and then convert back `fromString` safely.

    Swapi.Enum.Episode.NewHope
        |> Swapi.Enum.Episode.toString
        |> Swapi.Enum.Episode.fromString
        == Just NewHope

This can be useful for generating Strings to use for <select> menus to check which item was selected.

-}
fromString : String -> Maybe Users_select_column
fromString enumString =
    case enumString of
        "active" ->
            Just Active

        "admin" ->
            Just Admin

        "id" ->
            Just Id

        "username" ->
            Just Username

        _ ->
            Nothing