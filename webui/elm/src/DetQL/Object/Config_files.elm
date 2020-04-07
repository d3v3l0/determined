-- Do not manually edit this file, it was auto-generated by dillonkearns/elm-graphql
-- https://github.com/dillonkearns/elm-graphql


module DetQL.Object.Config_files exposing (..)

import CustomScalarCodecs
import DetQL.InputObject
import DetQL.Interface
import DetQL.Object
import DetQL.Scalar
import DetQL.Union
import Graphql.Internal.Builder.Argument as Argument exposing (Argument)
import Graphql.Internal.Builder.Object as Object
import Graphql.Internal.Encode as Encode exposing (Value)
import Graphql.Operation exposing (RootMutation, RootQuery, RootSubscription)
import Graphql.OptionalArgument exposing (OptionalArgument(..))
import Graphql.SelectionSet exposing (SelectionSet)
import Json.Decode as Decode


content : SelectionSet (Maybe CustomScalarCodecs.Bytea) DetQL.Object.Config_files
content =
    Object.selectionForField "(Maybe CustomScalarCodecs.Bytea)" "content" [] (CustomScalarCodecs.codecs |> DetQL.Scalar.unwrapCodecs |> .codecBytea |> .decoder |> Decode.nullable)


id : SelectionSet Int DetQL.Object.Config_files
id =
    Object.selectionForField "Int" "id" [] Decode.int